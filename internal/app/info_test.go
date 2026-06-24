package app

import (
	"context"
	"strings"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/testing/fake"
)

func TestShowInfo_ListServices(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub"},
		{ID: "notion", Name: "Notion"},
	})
	out, err := NewShowInfo(cat).Run(context.Background(), InfoInput{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.Markdown, "github") || !strings.Contains(out.Markdown, "notion") {
		t.Errorf("missing services: %s", out.Markdown)
	}
}

func TestShowInfo_Service(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub", Description: "GH API", Actions: []core.Action{
			{ID: "issues.read", Service: "github", Description: "read issue", Permission: "issues.read"},
		}},
	})
	out, err := NewShowInfo(cat).Run(context.Background(), InfoInput{Service: "github"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.Markdown, "issues.read") {
		t.Errorf("missing action: %s", out.Markdown)
	}
}

func TestShowInfo_Service_SkillOwnsHeader(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub", Description: "GH API", Actions: []core.Action{
			{ID: "issues.read", Service: "github", Description: "read issue", Permission: "issues.read"},
		}},
	}).WithSkill("github", "# GitHub\n\nUse the GitHub REST API.")
	out, err := NewShowInfo(cat).Run(context.Background(), InfoInput{Service: "github"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// The generated header must not duplicate the skill's own H1.
	if strings.Count(out.Markdown, "# GitHub") != 1 {
		t.Errorf("expected exactly one '# GitHub' header, got:\n%s", out.Markdown)
	}
	if !strings.Contains(out.Markdown, "GitHub REST API") {
		t.Errorf("skill body missing: %s", out.Markdown)
	}
}

func TestShowInfo_Service_NoSkillGeneratesHeader(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub", Description: "GH API"},
	})
	out, err := NewShowInfo(cat).Run(context.Background(), InfoInput{Service: "github"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.Markdown, "# GitHub") {
		t.Errorf("expected generated '# GitHub' header, got:\n%s", out.Markdown)
	}
}

func TestShowInfo_Action(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{fixtureService()})
	out, err := NewShowInfo(cat).Run(context.Background(), InfoInput{Service: "github", Action: "issues.read"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.Markdown, "Inputs") || !strings.Contains(out.Markdown, "owner") {
		t.Errorf("missing inputs: %s", out.Markdown)
	}
}
