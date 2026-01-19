package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/prompts/new_project.md
var newProjectPromptTemplate string

//go:embed templates/prompts/new_work.md
var newWorkPromptTemplate string

//go:embed templates/prompts/setup_prompt.md
var setupPromptTemplate string

//go:embed templates/prompts/loop_prompt.md
var loopPromptTemplate string

//go:embed templates/prompts/readme.md
var readmeTemplate string

//go:embed templates/prompts/explore_repo.md
var exploreRepoPromptTemplate string

func renderNewProjectPrompt(projectName, requirements string) (string, error) {
	return renderTemplate("new_project", newProjectPromptTemplate, projectName, requirements)
}

func renderNewWorkPrompt(projectName, requirements, existingPRD, work string) (string, error) {
	tmpl, err := template.New("new_work").Option("missingkey=error").Parse(newWorkPromptTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		ProjectName  string
		Requirements string
		ExistingPRD  string
		Work         string
	}{
		ProjectName:  projectName,
		Requirements: requirements,
		ExistingPRD:  existingPRD,
		Work:         work,
	}); err != nil {
		return "", err
	}
	out := buf.String()
	if out == "" {
		return "", fmt.Errorf("rendered template %s is empty", "new_work")
	}
	return out, nil
}

func renderSetupPrompt(projectName, requirements string) (string, error) {
	return renderTemplate("setup_prompt", setupPromptTemplate, projectName, requirements)
}

func renderLoopPrompt(projectName, requirements string) (string, error) {
	return renderTemplate("loop_prompt", loopPromptTemplate, projectName, requirements)
}

func renderReadme(projectName, requirements string) (string, error) {
	return renderTemplate("readme", readmeTemplate, projectName, requirements)
}

func renderExploreRepoPrompt(projectName string) (string, error) {
	return renderTemplate("explore_repo", exploreRepoPromptTemplate, projectName, "")
}

func renderTemplate(name, tmplText, projectName, requirements string) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(tmplText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		ProjectName  string
		Requirements string
	}{
		ProjectName:  projectName,
		Requirements: requirements,
	}); err != nil {
		return "", err
	}

	out := buf.String()
	if out == "" {
		return "", fmt.Errorf("rendered template %s is empty", name)
	}

	return out, nil
}
