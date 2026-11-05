import config from "../config/queue";

export interface JobOptions {
  queue?: string;
  attempts?: number;
  /** Base delay (ms) for exponential backoff between attempts. */
  backoff?: number;
  /** Cron pattern — makes this a scheduled job (≈ Laravel's scheduler). */
  schedule?: string;
}

export interface Job<T = any> {
  name: string;
  handle: (data: T) => Promise<void> | void;
  options: JobOptions;
}

/** Define a background job (≈ a Laravel Job class). */
export function job<T = void>(name: string, handle: Job<T>["handle"], options: JobOptions = {}): Job<T> {
  return { name, handle, options };
}

const queues = new Map<string, any>();

// Imported dynamically so projects on the "sync" driver don't need BullMQ installed.
const load = (pkg: string): Promise<any> => import(pkg);

async function queue(name: string) {
  if (!queues.has(name)) {
    const { Queue } = await load("bullmq");
    queues.set(name, new Queue(name, { connection: { url: config.redis } }));
  }
  return queues.get(name);
}

/** Push a job onto the queue (or run it inline with QUEUE_CONNECTION=sync). */
export async function dispatch<T>(job: Job<T>, data: T): Promise<void> {
  if (config.connection === "sync") return void (await job.handle(data));
  const q = await queue(job.options.queue ?? "default");
  await q.add(job.name, data, {
    attempts: job.options.attempts ?? 3,
    backoff: { type: "exponential", delay: job.options.backoff ?? 2000 },
    removeOnComplete: 1000,
    removeOnFail: 5000,
  });
}

/** Start workers for every registered job (≈ `php artisan queue:work`). */
export async function work(jobs: Job[], concurrency = 5) {
  if (config.connection === "sync") {
    console.log("QUEUE_CONNECTION=sync — jobs run inline, no worker needed.");
    return;
  }
  const { Worker } = await load("bullmq");
  const byQueue = Map.groupBy(jobs, (j) => j.options.queue ?? "default");

  for (const [name, list] of byQueue) {
    const handlers = new Map(list.map((j) => [j.name, j]));
    const worker = new Worker(
      name,
      async (task: { name: string; data: unknown }) => {
        const j = handlers.get(task.name);
        if (!j) throw new Error(`No handler for job "${task.name}"`);
        await j.handle(task.data);
      },
      { connection: { url: config.redis }, concurrency },
    );
    worker.on("completed", (task: any) => console.log(`✓ ${name}:${task.name} #${task.id}`));
    worker.on("failed", (task: any, err: Error) => console.error(`✗ ${name}:${task?.name} #${task?.id} ${err.message}`));

    for (const j of list.filter((j) => j.options.schedule)) {
      await (await queue(name)).upsertJobScheduler(j.name, { pattern: j.options.schedule }, { name: j.name });
    }
    console.log(`Worker listening on "${name}" (${list.map((j) => j.name).join(", ")})`);
  }
}
