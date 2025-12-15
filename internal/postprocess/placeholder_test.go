package postprocess

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWrapPlaceholders tests the wrapPlaceholders function
func TestWrapPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple placeholder",
			input:    "- SAML 인증 제공자 호스트 주소 : {saml Application url:port}",
			expected: "- SAML 인증 제공자 호스트 주소 : `{saml Application url:port}`",
		},
		{
			name:     "already wrapped with backticks",
			input:    "- SAML 인증 제공자 호스트 주소 : `{saml Application url:port}`",
			expected: "- SAML 인증 제공자 호스트 주소 : `{saml Application url:port}`",
		},
		{
			name:     "multiple placeholders",
			input:    "Use {username} and {password} to login",
			expected: "Use `{username}` and `{password}` to login",
		},
		{
			name:     "mixed wrapped and unwrapped",
			input:    "Use `{username}` and {password} to login",
			expected: "Use `{username}` and `{password}` to login",
		},
		{
			name:     "no placeholders",
			input:    "This is plain text without any placeholders",
			expected: "This is plain text without any placeholders",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "placeholder at start",
			input:    "{start} of line",
			expected: "`{start}` of line",
		},
		{
			name:     "placeholder at end",
			input:    "end of {line}",
			expected: "end of `{line}`",
		},
		{
			name:     "multiline with placeholders",
			input:    "Line 1: {value1}\nLine 2: {value2}",
			expected: "Line 1: `{value1}`\nLine 2: `{value2}`",
		},
		{
			name:     "nested braces in placeholder",
			input:    "Config: {host:port}",
			expected: "Config: `{host:port}`",
		},
		{
			name:     "unclosed brace",
			input:    "Unclosed {brace here",
			expected: "Unclosed {brace here",
		},
		{
			name:     "brace spans multiple lines - should not wrap",
			input:    "Start {brace\nend}",
			expected: "Start {brace\nend}",
		},
		{
			name: "simple code block - should not wrap placeholders inside",
			input: `Before {wrap-me}
` + "```" + `
{do-not-wrap}
` + "```" + `
After {wrap-me-too}`,
			expected: `Before ` + "`{wrap-me}`" + `
` + "```" + `
{do-not-wrap}
` + "```" + `
After ` + "`{wrap-me-too}`",
		},
		{
			name: "code block with language specifier",
			input: `Example:
` + "```json" + `
{
  "name": "{username}",
  "port": "{port}"
}
` + "```" + `
Use {config-path} for configuration.`,
			expected: `Example:
` + "```json" + `
{
  "name": "{username}",
  "port": "{port}"
}
` + "```" + `
Use ` + "`{config-path}`" + ` for configuration.`,
		},
		{
			name: "multiple code blocks",
			input: `First placeholder: {first}

` + "```bash" + `
export HOST={hostname}
export PORT={port}
` + "```" + `

Between blocks: {middle}

` + "```yaml" + `
config:
  url: {url}
  token: {token}
` + "```" + `

Last placeholder: {last}`,
			expected: `First placeholder: ` + "`{first}`" + `

` + "```bash" + `
export HOST={hostname}
export PORT={port}
` + "```" + `

Between blocks: ` + "`{middle}`" + `

` + "```yaml" + `
config:
  url: {url}
  token: {token}
` + "```" + `

Last placeholder: ` + "`{last}`",
		},
		{
			name:     "code block at start of content",
			input:    "```\n{inside}\n```\n{outside}",
			expected: "```\n{inside}\n```\n`{outside}`",
		},
		{
			name:     "code block at end of content",
			input:    "{outside}\n```\n{inside}\n```",
			expected: "`{outside}`\n```\n{inside}\n```",
		},
		{
			name:     "empty code block",
			input:    "{before}\n```\n```\n{after}",
			expected: "`{before}`\n```\n```\n`{after}`",
		},
		{
			name: "nested braces in code block",
			input: `Config: {wrap-this}
` + "```go" + `
func main() {
    data := map[string]string{
        "key": "{value}",
    }
    fmt.Println(data)
}
` + "```" + `
End: {wrap-this-too}`,
			expected: `Config: ` + "`{wrap-this}`" + `
` + "```go" + `
func main() {
    data := map[string]string{
        "key": "{value}",
    }
    fmt.Println(data)
}
` + "```" + `
End: ` + "`{wrap-this-too}`",
		},
		{
			name: "real world SAML documentation example",
			input: `# SAML 설정 가이드

## 설정 값

- 호스트 주소: {saml-host}
- 포트: {saml-port}

## 설정 예시

` + "```xml" + `
<EntityDescriptor entityID="{entity-id}">
  <SPSSODescriptor>
    <AssertionConsumerService
      Location="{acs-url}"
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"/>
  </SPSSODescriptor>
</EntityDescriptor>
` + "```" + `

## 환경 변수

` + "```bash" + `
SAML_ENTITY_ID={your-entity-id}
SAML_ACS_URL={your-acs-url}
SAML_CALLBACK={callback-url}
` + "```" + `

설정이 완료되면 {admin-url}에서 확인하세요.`,
			expected: `# SAML 설정 가이드

## 설정 값

- 호스트 주소: ` + "`{saml-host}`" + `
- 포트: ` + "`{saml-port}`" + `

## 설정 예시

` + "```xml" + `
<EntityDescriptor entityID="{entity-id}">
  <SPSSODescriptor>
    <AssertionConsumerService
      Location="{acs-url}"
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"/>
  </SPSSODescriptor>
</EntityDescriptor>
` + "```" + `

## 환경 변수

` + "```bash" + `
SAML_ENTITY_ID={your-entity-id}
SAML_ACS_URL={your-acs-url}
SAML_CALLBACK={callback-url}
` + "```" + `

설정이 완료되면 ` + "`{admin-url}`" + `에서 확인하세요.`,
		},
		{
			name:     "code block with already wrapped placeholders outside",
			input:    "Already wrapped: `{wrapped}`\n```\n{in-code}\n```\nNot wrapped: {not-wrapped}",
			expected: "Already wrapped: `{wrapped}`\n```\n{in-code}\n```\nNot wrapped: `{not-wrapped}`",
		},
		{
			name:     "consecutive code blocks",
			input:    "{before}\n```\n{code1}\n```\n```\n{code2}\n```\n{after}",
			expected: "`{before}`\n```\n{code1}\n```\n```\n{code2}\n```\n`{after}`",
		},
		// Markdown link tests - placeholders inside link paths should NOT be wrapped
		{
			name:     "image link with placeholder in path",
			input:    "![alt](files/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)",
			expected: "![alt](files/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)",
		},
		{
			name:     "markdown link with placeholder in path",
			input:    "[link text](path/to/{file-id}/document.md)",
			expected: "[link text](path/to/{file-id}/document.md)",
		},
		{
			name:     "mixed: placeholder in text wrapped, in path not wrapped",
			input:    "See {config} at [link](path/{id}/file.md) for details",
			expected: "See `{config}` at [link](path/{id}/file.md) for details",
		},
		{
			name:     "multiple image links with placeholders",
			input:    "![img1](a/{id1}.png) and ![img2](b/{id2}.jpg)",
			expected: "![img1](a/{id1}.png) and ![img2](b/{id2}.jpg)",
		},
		{
			name:     "placeholder in link text should be wrapped",
			input:    "[{placeholder}](path/to/file.md)",
			expected: "[`{placeholder}`](path/to/file.md)",
		},
		{
			name:     "complex path with GUID-like placeholder",
			input:    "![scrum image](files/019a8f59-33bd-715e-a328-58d9fe60f760/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)",
			expected: "![scrum image](files/019a8f59-33bd-715e-a328-58d9fe60f760/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)",
		},
		{
			name: "real world example with image and other placeholders",
			input: `# SCRUM 2025년 11월 3주

이미지 설정: {config-value}

![스크린샷](files/019a8f59-33bd-715e-a328-58d9fe60f760/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)

설정값: {another-config}`,
			expected: `# SCRUM 2025년 11월 3주

이미지 설정: ` + "`{config-value}`" + `

![스크린샷](files/019a8f59-33bd-715e-a328-58d9fe60f760/{8B771614-71E9-4C9F-9E6F-B67C10963A91}.png)

설정값: ` + "`{another-config}`",
		},
		{
			name:     "link path with placeholder at start",
			input:    "[link]({path}/file.md)",
			expected: "[link]({path}/file.md)",
		},
		{
			name:     "link path with placeholder at end",
			input:    "[link](path/{filename})",
			expected: "[link](path/{filename})",
		},
		{
			name:     "multiple placeholders in link path",
			input:    "[link]({dir}/{subdir}/{file}.md)",
			expected: "[link]({dir}/{subdir}/{file}.md)",
		},
		{
			name:     "nested parentheses - link ends at first close paren",
			input:    "[link](path/{id}) and {wrap-me}",
			expected: "[link](path/{id}) and `{wrap-me}`",
		},
		{
			name:     "malformed link with newline should reset state",
			input:    "[link](path/{id}\n{wrap-me}",
			expected: "[link](path/{id}\n`{wrap-me}`",
		},
		// JSON-like patterns should NOT be wrapped
		{
			name:     "JSON pattern with double quotes",
			input:    `"doc": { ... }`,
			expected: `"doc": { ... }`,
		},
		{
			name:     "JSON pattern with spaces around colon",
			input:    `"doc" : { ... }`,
			expected: `"doc" : { ... }`,
		},
		{
			name:     "JSON pattern with single quotes",
			input:    `'doc': { ... }`,
			expected: `'doc': { ... }`,
		},
		{
			name:     "JSON nested object",
			input:    `"config": {"nested": "value"}`,
			expected: `"config": {"nested": "value"}`,
		},
		{
			name:     "JSON pattern in markdown text",
			input:    `다음과 같이 설정합니다: "doc": { ... } 형식으로`,
			expected: `다음과 같이 설정합니다: "doc": { ... } 형식으로`,
		},
		{
			name:     "mixed JSON and placeholder",
			input:    `"config": {value} and use {placeholder} here`,
			expected: `"config": {value} and use ` + "`{placeholder}`" + ` here`,
		},
		{
			name:     "JSON pattern without quotes should wrap",
			input:    `config: {value}`,
			expected: `config: ` + "`{value}`",
		},
		{
			name:     "multiple JSON patterns",
			input:    `"a": {x} and "b": {y}`,
			expected: `"a": {x} and "b": {y}`,
		},
		{
			name:     "JSON array value should still wrap",
			input:    `"items": [{item}]`,
			expected: `"items": [` + "`{item}`" + `]`,
		},
		{
			name: "real world JSON example",
			input: `API 응답 형식:

"doc": { ... }

설정값: {config-value}`,
			expected: `API 응답 형식:

"doc": { ... }

설정값: ` + "`{config-value}`",
		},
		// Inline code tests - placeholders inside inline code should NOT be wrapped
		{
			name:     "placeholder with $ prefix inside inline code",
			input:    "  - **SharePoint:** `https://login.microsoftonline.com/login.srf?wa=wsignin1%2E0&rver=6%2E1%2E6206%2E0&wreply=https%3A%2F%2F<tenant>.sharepoint.com%2F&whr=${federated-domain}`",
			expected: "  - **SharePoint:** `https://login.microsoftonline.com/login.srf?wa=wsignin1%2E0&rver=6%2E1%2E6206%2E0&wreply=https%3A%2F%2F<tenant>.sharepoint.com%2F&whr=${federated-domain}`",
		},
		{
			name:     "simple placeholder inside inline code",
			input:    "Use `{placeholder}` for config",
			expected: "Use `{placeholder}` for config",
		},
		{
			name:     "multiple placeholders inside inline code",
			input:    "URL: `https://{host}:{port}/{path}`",
			expected: "URL: `https://{host}:{port}/{path}`",
		},
		{
			name:     "mixed inline code and regular placeholder",
			input:    "Use `{inside}` and {outside} in text",
			expected: "Use `{inside}` and `{outside}` in text",
		},
		{
			name:     "placeholder after inline code",
			input:    "Code `example` then {placeholder}",
			expected: "Code `example` then `{placeholder}`",
		},
		{
			name:     "placeholder before inline code",
			input:    "{placeholder} then `example`",
			expected: "`{placeholder}` then `example`",
		},
		{
			name:     "nested backticks edge case",
			input:    "``{inside}`` and {outside}",
			expected: "``{inside}`` and `{outside}`",
		},
		// Various ${...} patterns inside inline code
		{
			name:     "dollar brace with simple word",
			input:    "Use `${variable}` in config",
			expected: "Use `${variable}` in config",
		},
		{
			name:     "dollar brace with hyphenated word",
			input:    "Set `${my-variable}` here",
			expected: "Set `${my-variable}` here",
		},
		{
			name:     "dollar brace with underscore",
			input:    "Use `${my_var}` for env",
			expected: "Use `${my_var}` for env",
		},
		{
			name:     "dollar brace with uppercase",
			input:    "Export `${HOME}` path",
			expected: "Export `${HOME}` path",
		},
		{
			name:     "dollar brace with mixed case",
			input:    "Set `${myVariable}` value",
			expected: "Set `${myVariable}` value",
		},
		{
			name:     "dollar brace with numbers",
			input:    "Port `${PORT_8080}` is used",
			expected: "Port `${PORT_8080}` is used",
		},
		{
			name:     "dollar brace with dots",
			input:    "Config `${app.config.value}` here",
			expected: "Config `${app.config.value}` here",
		},
		{
			name:     "dollar brace with colons",
			input:    "Default `${VAR:-default}` value",
			expected: "Default `${VAR:-default}` value",
		},
		{
			name:     "multiple dollar braces in inline code",
			input:    "URL `${protocol}://${host}:${port}/${path}`",
			expected: "URL `${protocol}://${host}:${port}/${path}`",
		},
		{
			name:     "dollar brace in URL inline code",
			input:    "`https://example.com?token=${TOKEN}`",
			expected: "`https://example.com?token=${TOKEN}`",
		},
		{
			name:     "dollar brace with special bash syntax",
			input:    "Expand `${!indirect}` variable",
			expected: "Expand `${!indirect}` variable",
		},
		{
			name:     "dollar brace with bash array",
			input:    "Array `${arr[@]}` expansion",
			expected: "Array `${arr[@]}` expansion",
		},
		{
			name:     "dollar brace with bash substring",
			input:    "Substring `${var:0:5}` operation",
			expected: "Substring `${var:0:5}` operation",
		},
		{
			name:     "dollar brace with bash replacement",
			input:    "Replace `${var//old/new}` pattern",
			expected: "Replace `${var//old/new}` pattern",
		},
		{
			name:     "dollar brace outside inline code wraps only braces",
			input:    "Use ${variable} in config",
			expected: "Use $`{variable}` in config",
		},
		{
			name:     "mixed dollar brace inside and outside",
			input:    "Inside `${inside}` and outside ${outside}",
			expected: "Inside `${inside}` and outside $`{outside}`",
		},
		{
			name:     "real SharePoint URL with tenant and federated-domain",
			input:    "  - **SharePoint:** `https://login.microsoftonline.com/login.srf?wa=wsignin1%2E0&rver=6%2E1%2E6206%2E0&wreply=https%3A%2F%2F${tenant}.sharepoint.com%2F&whr=${federated-domain}`",
			expected: "  - **SharePoint:** `https://login.microsoftonline.com/login.srf?wa=wsignin1%2E0&rver=6%2E1%2E6206%2E0&wreply=https%3A%2F%2F${tenant}.sharepoint.com%2F&whr=${federated-domain}`",
		},
		{
			name:     "Azure AD URL pattern",
			input:    "`https://login.microsoftonline.com/${tenantId}/oauth2/v2.0/authorize`",
			expected: "`https://login.microsoftonline.com/${tenantId}/oauth2/v2.0/authorize`",
		},
		{
			name:     "Kubernetes env var pattern",
			input:    "Value: `${SECRET_KEY_REF}`",
			expected: "Value: `${SECRET_KEY_REF}`",
		},
		{
			name:     "Docker compose variable",
			input:    "Port: `${DOCKER_PORT:-3000}`",
			expected: "Port: `${DOCKER_PORT:-3000}`",
		},
		{
			name:     "GitHub Actions variable",
			input:    "Token: `${secrets.GITHUB_TOKEN}`",
			expected: "Token: `${secrets.GITHUB_TOKEN}`",
		},
		{
			name:     "empty dollar brace",
			input:    "Empty `${}` pattern",
			expected: "Empty `${}` pattern",
		},
		{
			name:     "dollar brace with space inside",
			input:    "Spaced `${my var}` pattern",
			expected: "Spaced `${my var}` pattern",
		},
		{
			name:     "dollar brace with Korean",
			input:    "Korean `${한글변수}` pattern",
			expected: "Korean `${한글변수}` pattern",
		},
		{
			name:     "complex URL with multiple variables",
			input:    "`https://${subdomain}.${domain}.com:${port}/${path}?key=${apiKey}&secret=${apiSecret}`",
			expected: "`https://${subdomain}.${domain}.com:${port}/${path}?key=${apiKey}&secret=${apiSecret}`",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapPlaceholders(tc.input)
			if result != tc.expected {
				t.Errorf("wrapPlaceholders(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestWrapAngleBrackets tests the wrapAngleBrackets function
func TestWrapAngleBrackets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple empty angle brackets",
			input:    "React uses <> for fragments",
			expected: "React uses `<>` for fragments",
		},
		{
			name:     "closing fragment tag",
			input:    "End with </>",
			expected: "End with `</>`",
		},
		{
			name:     "both opening and closing fragments",
			input:    "Use <> and </> in React",
			expected: "Use `<>` and `</>` in React",
		},
		{
			name:     "already wrapped with backticks - opening",
			input:    "Use `<>` for fragments",
			expected: "Use `<>` for fragments",
		},
		{
			name:     "already wrapped with backticks - closing",
			input:    "End with `</>`",
			expected: "End with `</>`",
		},
		{
			name:     "mixed wrapped and unwrapped",
			input:    "Use `<>` and </> together",
			expected: "Use `<>` and `</>` together",
		},
		{
			name:     "no angle brackets",
			input:    "This is plain text without any angle brackets",
			expected: "This is plain text without any angle brackets",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "angle brackets at start",
			input:    "<> starts the line",
			expected: "`<>` starts the line",
		},
		{
			name:     "angle brackets at end",
			input:    "end with <>",
			expected: "end with `<>`",
		},
		{
			name:     "multiline with angle brackets",
			input:    "Line 1: <>\nLine 2: </>",
			expected: "Line 1: `<>`\nLine 2: `</>`",
		},
		{
			name:     "normal HTML tags should not be affected",
			input:    "Use <div> and </div> for HTML",
			expected: "Use <div> and </div> for HTML",
		},
		{
			name:     "self-closing tags should not be affected",
			input:    "Use <br/> and <img/>",
			expected: "Use <br/> and <img/>",
		},
		{
			name: "inside code block - should not wrap",
			input: `Before <>
` + "```jsx" + `
return (
  <>
    <Component />
  </>
);
` + "```" + `
After <>`,
			expected: `Before ` + "`<>`" + `
` + "```jsx" + `
return (
  <>
    <Component />
  </>
);
` + "```" + `
After ` + "`<>`",
		},
		{
			name:     "inside inline code - should not wrap",
			input:    "Use `<>` and regular <> in text",
			expected: "Use `<>` and regular `<>` in text",
		},
		{
			name:     "multiple inline code blocks",
			input:    "Compare `<>` with `</>` and <> with </>",
			expected: "Compare `<>` with `</>` and `<>` with `</>`",
		},
		{
			name: "real world React documentation example",
			input: `# React Fragments

React 에서 여러 요소를 반환할 때 <> 와 </> 를 사용합니다.

## 예시

` + "```jsx" + `
function App() {
  return (
    <>
      <Header />
      <Main />
      <Footer />
    </>
  );
}
` + "```" + `

<> 는 Fragment 의 단축 문법입니다.`,
			expected: `# React Fragments

React 에서 여러 요소를 반환할 때 ` + "`<>`" + ` 와 ` + "`</>`" + ` 를 사용합니다.

## 예시

` + "```jsx" + `
function App() {
  return (
    <>
      <Header />
      <Main />
      <Footer />
    </>
  );
}
` + "```" + `

` + "`<>`" + ` 는 Fragment 의 단축 문법입니다.`,
		},
		{
			name:     "consecutive angle brackets",
			input:    "<><></>",
			expected: "`<>``<>``</>`",
		},
		{
			name:     "angle brackets with surrounding text",
			input:    "text<>more</>end",
			expected: "text`<>`more`</>`end",
		},
		{
			name:     "single less than sign",
			input:    "a < b and c > d",
			expected: "a < b and c > d",
		},
		{
			name:     "comparison operators should not be affected",
			input:    "if (x < y && y > z)",
			expected: "if (x < y && y > z)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapAngleBrackets(tc.input)
			if result != tc.expected {
				t.Errorf("wrapAngleBrackets(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestWrapAngleBracketsWithBackticks_Integration tests the full function with files
func TestWrapAngleBracketsWithBackticks_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-wrap-angle-brackets-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file
	testFile := filepath.Join(tempDir, "test.md")
	content := `# React Fragments

Use <> and </> for fragments.
Already wrapped: ` + "`<>`" + `

` + "```jsx" + `
return <><Child /></>;
` + "```" + `
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run function
	if err := WrapAngleBracketsWithBackticks(tempDir); err != nil {
		t.Fatalf("WrapAngleBracketsWithBackticks failed: %v", err)
	}

	// Read result
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}

	expected := `# React Fragments

Use ` + "`<>`" + ` and ` + "`</>`" + ` for fragments.
Already wrapped: ` + "`<>`" + `

` + "```jsx" + `
return <><Child /></>;
` + "```" + `
`
	if string(result) != expected {
		t.Errorf("file content mismatch.\nGot:\n%s\nExpected:\n%s", string(result), expected)
	}
}

// TestWrapRawHTML tests the wrapRawHTML function
func TestWrapRawHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple table",
			input: `# 4. 응답 데이터

<table style="min-width: 934px"><tbody><tr><th colspan="1" rowspan="1" colwidth="200"><p>필드</p></th><th colspan="1" rowspan="1" colwidth="92"><p>타입</p></th><th colspan="1" rowspan="1" colwidth="617"><p>설명</p></th></tr></tbody></table>

#5. 테스트`,
			expected: "# 4. 응답 데이터\n\n```html\n<table style=\"min-width: 934px\"><tbody><tr><th colspan=\"1\" rowspan=\"1\" colwidth=\"200\"><p>필드</p></th><th colspan=\"1\" rowspan=\"1\" colwidth=\"92\"><p>타입</p></th><th colspan=\"1\" rowspan=\"1\" colwidth=\"617\"><p>설명</p></th></tr></tbody></table>\n```\n\n#5. 테스트",
		},
		{
			name: "multiline table",
			input: `# 응답 형식

<table>
<tbody>
<tr>
<th>필드</th>
<td>값</td>
</tr>
</tbody>
</table>

설명 텍스트`,
			expected: "# 응답 형식\n\n```html\n<table>\n<tbody>\n<tr>\n<th>필드</th>\n<td>값</td>\n</tr>\n</tbody>\n</table>\n```\n\n설명 텍스트",
		},
		{
			name: "table already in code block",
			input: "# 예시\n\n```html\n<table><tr><td>data</td></tr></table>\n```\n\n끝",
			expected: "# 예시\n\n```html\n<table><tr><td>data</td></tr></table>\n```\n\n끝",
		},
		{
			name:     "no HTML content",
			input:    "# 제목\n\n일반 텍스트입니다.",
			expected: "# 제목\n\n일반 텍스트입니다.",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name: "table at start of file",
			input: `<table><tr><td>시작</td></tr></table>

이후 텍스트`,
			expected: "```html\n<table><tr><td>시작</td></tr></table>\n```\n\n이후 텍스트",
		},
		{
			name: "table at end of file",
			input: `텍스트

<table><tr><td>끝</td></tr></table>`,
			expected: "텍스트\n\n```html\n<table><tr><td>끝</td></tr></table>\n```",
		},
		{
			name: "multiple separate tables",
			input: `# 첫 번째 테이블

<table><tr><td>1</td></tr></table>

중간 텍스트

<table><tr><td>2</td></tr></table>

끝`,
			expected: "# 첫 번째 테이블\n\n```html\n<table><tr><td>1</td></tr></table>\n```\n\n중간 텍스트\n\n```html\n<table><tr><td>2</td></tr></table>\n```\n\n끝",
		},
		{
			name: "table with thead",
			input: `<table>
<thead>
<tr><th>헤더</th></tr>
</thead>
<tbody>
<tr><td>데이터</td></tr>
</tbody>
</table>`,
			expected: "```html\n<table>\n<thead>\n<tr><th>헤더</th></tr>\n</thead>\n<tbody>\n<tr><td>데이터</td></tr>\n</tbody>\n</table>\n```",
		},
		{
			name: "mixed content with code block in between",
			input: `# 문서

<table><tr><td>표1</td></tr></table>

` + "```javascript" + `
const x = 1;
` + "```" + `

<table><tr><td>표2</td></tr></table>`,
			expected: "# 문서\n\n```html\n<table><tr><td>표1</td></tr></table>\n```\n\n```javascript\nconst x = 1;\n```\n\n```html\n<table><tr><td>표2</td></tr></table>\n```",
		},
		{
			name: "real world API documentation example",
			input: `# 4. 응답 데이터

<table style="min-width: 934px"><tbody><tr><th colspan="1" rowspan="1" colwidth="200"><p>필드</p></th><th colspan="1" rowspan="1" colwidth="92"><p>타입</p></th><th colspan="1" rowspan="1" colwidth="617"><p>설명</p></th></tr><tr><td colspan="1" rowspan="1" colwidth="200"><p>resultCode</p></td><td colspan="1" rowspan="1" colwidth="92"><p>String</p></td><td colspan="1" rowspan="1" colwidth="617"><p>응답 코드</p></td></tr></tbody></table>

#5. 테스트

일반 텍스트입니다.`,
			expected: "# 4. 응답 데이터\n\n```html\n<table style=\"min-width: 934px\"><tbody><tr><th colspan=\"1\" rowspan=\"1\" colwidth=\"200\"><p>필드</p></th><th colspan=\"1\" rowspan=\"1\" colwidth=\"92\"><p>타입</p></th><th colspan=\"1\" rowspan=\"1\" colwidth=\"617\"><p>설명</p></th></tr><tr><td colspan=\"1\" rowspan=\"1\" colwidth=\"200\"><p>resultCode</p></td><td colspan=\"1\" rowspan=\"1\" colwidth=\"92\"><p>String</p></td><td colspan=\"1\" rowspan=\"1\" colwidth=\"617\"><p>응답 코드</p></td></tr></tbody></table>\n```\n\n#5. 테스트\n\n일반 텍스트입니다.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapRawHTML(tc.input)
			if result != tc.expected {
				t.Errorf("wrapRawHTML() mismatch\nInput:\n%s\n\nGot:\n%s\n\nExpected:\n%s", tc.input, result, tc.expected)
			}
		})
	}
}

// TestWrapRawHTMLWithCodeBlock_Integration tests the full function with files
func TestWrapRawHTMLWithCodeBlock_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-wrap-raw-html-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file
	testFile := filepath.Join(tempDir, "test.md")
	content := `# API 응답

<table><tbody><tr><th>필드</th><td>값</td></tr></tbody></table>

설명
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run function
	if err := WrapRawHTMLWithCodeBlock(tempDir); err != nil {
		t.Fatalf("WrapRawHTMLWithCodeBlock failed: %v", err)
	}

	// Read result
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}

	expected := "# API 응답\n\n```html\n<table><tbody><tr><th>필드</th><td>값</td></tr></tbody></table>\n```\n\n설명\n"
	if string(result) != expected {
		t.Errorf("file content mismatch.\nGot:\n%s\nExpected:\n%s", string(result), expected)
	}
}

// TestWrapPlaceholdersWithBackticks_Integration tests the full function with files
func TestWrapPlaceholdersWithBackticks_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-wrap-placeholders-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file
	testFile := filepath.Join(tempDir, "test.md")
	content := `# Configuration

- SAML 인증 제공자 호스트 주소 : {saml Application url:port}
- Already wrapped: ` + "`{wrapped}`" + `
- Another placeholder: {another}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run function
	if err := WrapPlaceholdersWithBackticks(tempDir); err != nil {
		t.Fatalf("WrapPlaceholdersWithBackticks failed: %v", err)
	}

	// Read result
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read result file: %v", err)
	}

	expected := `# Configuration

- SAML 인증 제공자 호스트 주소 : ` + "`{saml Application url:port}`" + `
- Already wrapped: ` + "`{wrapped}`" + `
- Another placeholder: ` + "`{another}`" + `
`
	if string(result) != expected {
		t.Errorf("file content mismatch.\nGot:\n%s\nExpected:\n%s", string(result), expected)
	}
}
