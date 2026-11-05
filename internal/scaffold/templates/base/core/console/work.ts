// Invoked by `forge queue:work`.
import jobs from "../../app/jobs";
import { work } from "../queue";

const concurrency = Number(process.argv[process.argv.indexOf("--concurrency") + 1]) || 5;
await work(jobs, concurrency);
