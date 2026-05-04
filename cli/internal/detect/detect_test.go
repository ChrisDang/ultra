package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Meta-framework tests ---

func TestDetectNextJS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectNextJSTS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.ts", "export default {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
}

func TestDetectNuxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nuxt.config.ts", "export default {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nuxt")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSvelteKit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "svelte.config.js", "export default {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "sveltekit")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectAstro(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "astro.config.mjs", "export default {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "astro")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRemixConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "remix.config.js", "module.exports = {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRemixDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"@remix-run/node":"^2.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
}

func TestDetectGatsby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "gatsby-config.js", "module.exports = {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "gatsby")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectAngular(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "angular.json", "{}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "angular")
	assertContains(t, stack.Providers, "vercel")
}

// --- Library tests ---

func TestDetectReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "react")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectVue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"vue":"^3.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "vue")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSvelte(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"svelte":"^4.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "svelte")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectSolid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"solid-js":"^1.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "solid")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectExpo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"expo":"^50.0"}}`)
	writeFile(t, dir, "app.json", `{"expo":{}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "expo")
	assertContains(t, stack.Providers, "expo")
}

// --- Meta-framework de-duplication ---

func TestNextJSSuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertNotContains(t, stack.Frameworks, "react")
}

func TestNuxtSuppressesVue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nuxt.config.ts", "export default {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"vue":"^3.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nuxt")
	assertNotContains(t, stack.Frameworks, "vue")
}

func TestSvelteKitSuppressesSvelte(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "svelte.config.js", "export default {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"svelte":"^4.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "sveltekit")
	assertNotContains(t, stack.Frameworks, "svelte")
}

func TestRemixSuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "remix.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0","@remix-run/node":"^2.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "remix")
	assertNotContains(t, stack.Frameworks, "react")
}

func TestGatsbySuppressesReact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "gatsby-config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "gatsby")
	assertNotContains(t, stack.Frameworks, "react")
}

// --- Language tests ---

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "go")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectRust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "rust")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==2.0")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "python")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPythonPyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "python")
}

func TestDetectRuby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "ruby")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectPHP(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", "{}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "php")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectJavaMaven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectJavaGradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "apply plugin: 'java'")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
}

func TestDetectJavaGradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "plugins { java }")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "java")
}

func TestDetectScala(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.sbt", `name := "app"`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "scala")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectDeno(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deno.json", "{}")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "deno")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectBun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bunfig.toml", "[install]")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "bun")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectNodeFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"my-app"}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "node")
	assertContains(t, stack.Providers, "vercel")
}

func TestNodeFallbackSuppressedByFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	stack := DetectStack(dir)
	assertNotContains(t, stack.Frameworks, "node")
}

// --- Build/deploy indicator tests ---

func TestDetectDocker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM node:18")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "docker")
	assertContains(t, stack.Providers, "vercel")
}

func TestDetectVercelJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "vercel.json", `{"version": 2}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "vercel")
	if len(stack.Frameworks) != 0 {
		t.Errorf("expected no frameworks for bare vercel.json, got %v", stack.Frameworks)
	}
}

func TestDetectStaticHTML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html></html>")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "static")
	assertContains(t, stack.Providers, "vercel")
}

func TestStaticSuppressedByFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "index.html", "<html></html>")
	stack := DetectStack(dir)
	assertNotContains(t, stack.Frameworks, "static")
}

// --- Infrastructure tests ---

func TestDetectSupabaseDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	mkdir(t, dir, "supabase")
	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "vercel")
	assertContains(t, stack.Providers, "supabase")
}

func TestDetectEasJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "eas.json", "{}")
	stack := DetectStack(dir)
	assertContains(t, stack.Providers, "expo")
}

// --- Multi-framework (monorepo) tests ---

func TestMonorepoNextJSPlusGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "go.mod", "module example.com/app")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Frameworks, "go")
	assertNotContains(t, stack.Frameworks, "react")
	assertContains(t, stack.Providers, "vercel")
}

func TestMonorepoNextJSPlusGoPlusSupabase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "go.mod", "module example.com/app")
	mkdir(t, dir, "supabase")
	stack := DetectStack(dir)
	assertContains(t, stack.Frameworks, "nextjs")
	assertContains(t, stack.Frameworks, "go")
	assertContains(t, stack.Providers, "vercel")
	assertContains(t, stack.Providers, "supabase")
}

// --- Supabase nudge tests ---

func TestNudgeDockerComposePostgres(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n    image: postgres:15")
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeEnvDatabaseURL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/mydb")
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgePrisma(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	mkdir(t, dir, "prisma")
	writeFile(t, dir, "prisma/schema.prisma", "generator client {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeDrizzle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "drizzle.config.ts", "export default {}")
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgePgDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"pg":"^8.0"}}`)
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNudgeSQLFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	mkdir(t, dir, "migrations")
	writeFile(t, dir, "migrations/001_init.sql", "CREATE TABLE users();")
	stack := DetectStack(dir)
	assertContains(t, stack.Nudges, "supabase")
}

func TestNoNudgeWhenSupabaseExists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n    image: postgres:15")
	mkdir(t, dir, "supabase")
	stack := DetectStack(dir)
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges when supabase/ exists, got %v", stack.Nudges)
	}
}

func TestNoNudgeWhenNoDatabaseIndicators(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app")
	stack := DetectStack(dir)
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges, got %v", stack.Nudges)
	}
}

// --- Edge cases ---

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	stack := DetectStack(dir)
	if len(stack.Frameworks) != 0 {
		t.Errorf("expected no frameworks, got %v", stack.Frameworks)
	}
	if len(stack.Providers) != 0 {
		t.Errorf("expected no providers, got %v", stack.Providers)
	}
	if len(stack.Nudges) != 0 {
		t.Errorf("expected no nudges, got %v", stack.Nudges)
	}
}

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, slice []string, val string) {
	t.Helper()
	for _, v := range slice {
		if v == val {
			return
		}
	}
	t.Errorf("expected %v to contain %q", slice, val)
}

func assertNotContains(t *testing.T, slice []string, val string) {
	t.Helper()
	for _, v := range slice {
		if v == val {
			t.Errorf("expected %v NOT to contain %q", slice, val)
			return
		}
	}
}
