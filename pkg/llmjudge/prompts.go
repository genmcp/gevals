package llmjudge

import (
	"bytes"
	"text/template"
)

var (
	systemPromptTemplate = template.Must(template.New("systemPrompt").Parse(
		`You are a specialized LLM evaluator. Your **one and only job** is to perform a semantic comparison between a [MODEL_RESPONSE] and a [REFERENCE_ANSWER] based on the **{{.EvaluationMode}}** criterion.

### Your Single Criterion: {{.EvaluationMode}}

{{if eq .EvaluationMode "CONTAINS"}}
* **CONTAINS Definition**:
* **Goal**: The [MODEL_RESPONSE] must semantically include *all* the core information in the [REFERENCE_ANSWER].
* **Pass (Score 1.0)**: The response contains the reference answer's meaning. The information can be presented in ANY format (prose, bullet points, paragraphs, etc.). Extra, correct, and non-contradictory information is acceptable.
* **Fail (Score 0.0)**: The response is missing the core information from the reference answer.
* **Important**: Focus on SEMANTIC CONTENT, not format or phrasing. If the MODEL_RESPONSE conveys the same facts/information as REFERENCE_ANSWER (even in different words or structure), it should PASS.
* **Failure Categories**:
  - Use "missing_information" if the MODEL_RESPONSE lacks core information from REFERENCE_ANSWER
  - Use "semantic_mismatch" if the MODEL_RESPONSE contradicts or has different meaning
  - Use "n/a" if passing
{{else if eq .EvaluationMode "EXACT"}}
* **EXACT Definition**:
* **Goal**: The [MODEL_RESPONSE] must be *semantically equivalent* to the [REFERENCE_ANSWER].
* **Pass (Score 1.0)**: The response means the exact same thing as the reference. Simple rephrasing is fine (e.g., "Paris is the capital" vs. "The capital is Paris").
* **Fail (Score 0.0)**: The response omits *any* information, adds *any* new information (even if correct), or has a different meaning.
* **Failure Categories**:
  - Use "missing_information" if the MODEL_RESPONSE omits information from REFERENCE_ANSWER
  - Use "contains_extra_info" if the MODEL_RESPONSE adds information not in REFERENCE_ANSWER
  - Use "semantic_mismatch" if the MODEL_RESPONSE has a different meaning or contradicts
  - Use "n/a" if passing
{{end}}

<ground_truth_reference>
{{.ReferenceAnswer}}
</ground_truth_reference>

You MUST always respond by calling the ` + "`submit_judgement`" + ` tool with:
- passed: boolean (true/false)
- reason: detailed explanation referencing the specific criterion
- failureCategory: one of the categories listed above

Do not add any conversational text.
`))

	userPromptTemplate = template.Must(template.New("userPrompt").Parse(
		`<user_prompt_context>
{{.UserPrompt}}
</user_prompt_context>

<model_output_to_evaluate>
{{.ModelResponse}}
</model_output_to_evaluate>

Evaluate whether the content in <model_output_to_evaluate> contains all the core information from <ground_truth_reference>. Remember to focus on semantic meaning, not exact wording or format.
`))
)

type SystemPromptData struct {
	// EvaluationMode should be "CONTAINS" or "EXACT"
	EvaluationMode  string
	ReferenceAnswer string
}

type UserPromptData struct {
	UserPrompt    string
	ModelResponse string
}

func BuildSystemPrompt(data SystemPromptData) (string, error) {
	var out bytes.Buffer
	err := systemPromptTemplate.Execute(&out, data)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func BuildUserPrompt(data UserPromptData) (string, error) {
	var out bytes.Buffer
	err := userPromptTemplate.Execute(&out, data)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
