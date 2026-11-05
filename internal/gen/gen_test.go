package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madebyjonny/forge/internal/project"
)

func TestNames(t *testing.T) {
	cases := map[string]Names{
		"post":      {Name: "Post", Camel: "post", File: "post", Var: "posts", Table: "posts", Plural: "posts", Path: "/api/posts"},
		"BlogPost":  {Name: "BlogPost", Camel: "blogPost", File: "blog-post", Var: "blogPosts", Table: "blog_posts", Plural: "blog-posts", Path: "/api/blog-posts"},
		"category":  {Name: "Category", Camel: "category", File: "category", Var: "categories", Table: "categories", Plural: "categories", Path: "/api/categories"},
		"box":       {Name: "Box", Camel: "box", File: "box", Var: "boxes", Table: "boxes", Plural: "boxes", Path: "/api/boxes"},
		"person":    {Name: "Person", Camel: "person", File: "person", Var: "people", Table: "people", Plural: "people", Path: "/api/people"},
		"api_token": {Name: "ApiToken", Camel: "apiToken", File: "api-token", Var: "apiTokens", Table: "api_tokens", Plural: "api-tokens", Path: "/api/api-tokens"},
	}
	for in, want := range cases {
		got := NamesFor(in)
		got.Kebab = ""
		if got != want {
			t.Errorf("NamesFor(%q)\n got  %+v\n want %+v", in, got, want)
		}
	}
	if s := Strip(Pascal("PostController"), "Controller"); s != "Post" {
		t.Errorf("Strip = %q", s)
	}
}

func TestParseField(t *testing.T) {
	cases := []struct{ spec, column, rule string }{
		{"title:string", `title: t.string("title").notNull()`, `title: z.string().trim().min(1).max(255)`},
		{"body:text?", `body: t.text("body")`, `body: z.string().nullish()`},
		{"bio:text:nullable", `bio: t.text("bio")`, `bio: z.string().nullish()`},
		{"published:bool:default=false", `published: t.boolean("published").notNull().default(false)`, `published: z.boolean().optional()`},
		{"price:float:default=9.5", `price: t.float("price").notNull().default(9.5)`, `price: z.coerce.number().optional()`},
		{"email:string:unique", `email: t.string("email").notNull().unique()`, `email: z.string().trim().min(1).max(255)`},
		{"user:references", `userId: t.foreignId("user_id", () => users.id).notNull()`, `userId: z.string().min(1)`},
		{"author:references:user", `authorId: t.foreignId("author_id", () => users.id).notNull()`, `authorId: z.string().min(1)`},
		{"publishedAt:datetime?", `publishedAt: t.timestamp("published_at")`, `publishedAt: z.coerce.date().nullish()`},
	}
	for _, c := range cases {
		f, err := ParseField(c.spec)
		if err != nil {
			t.Fatalf("%s: %v", c.spec, err)
		}
		if got := f.ColumnCode(); got != c.column {
			t.Errorf("%s column:\n got  %s\n want %s", c.spec, got, c.column)
		}
		if got := f.RuleCode(); got != c.rule {
			t.Errorf("%s rule:\n got  %s\n want %s", c.spec, got, c.rule)
		}
	}
}

func TestParseFieldErrors(t *testing.T) {
	for _, spec := range []string{"title", "title:blob", "title:string:wat", ":string"} {
		if _, err := ParseField(spec); err == nil {
			t.Errorf("expected error for %q", spec)
		}
	}
	if _, err := ParseFields([]string{"a:string", "a:text"}); err == nil {
		t.Error("expected duplicate field error")
	}
}

func TestToolGenerator(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	write("app/mcp/index.ts", "import whoami from \"./tools/whoami\";\n// forge:imports\nexport default [\n  whoami,\n  // forge:tools\n];\n")
	write("app/models/blog-post.ts", "export const blogPosts = {}; // userId: owned\n")

	p := &project.Project{Root: root, Manifest: project.Manifest{Features: []string{"auth", "mcp"}}}
	g := New(p, false)
	if err := g.Tool("SendReportTool", false); err != nil {
		t.Fatal(err)
	}
	if err := g.Tool("BlogPost", true); err != nil {
		t.Fatal(err)
	}
	single, _ := os.ReadFile(filepath.Join(root, "app/mcp/tools/send-report.ts"))
	if !strings.Contains(string(single), `"send_report"`) {
		t.Errorf("tool name:\n%s", single)
	}
	crud, _ := os.ReadFile(filepath.Join(root, "app/mcp/tools/blog-post.ts"))
	for _, want := range []string{`"list_blog_posts"`, `"update_blog_post"`, "userId: user.id", "List blog posts"} {
		if !strings.Contains(string(crud), want) {
			t.Errorf("resource tools missing %q:\n%s", want, crud)
		}
	}
	index, _ := os.ReadFile(filepath.Join(root, "app/mcp/index.ts"))
	if !strings.Contains(string(index), "  sendReport,\n  ...blogPostTools,\n  // forge:tools") {
		t.Errorf("index:\n%s", index)
	}

	noMCP := New(&project.Project{Root: root, Manifest: project.Manifest{Features: []string{"auth"}}}, false)
	if err := noMCP.Tool("X", false); err == nil {
		t.Error("expected error without mcp feature")
	}
}
