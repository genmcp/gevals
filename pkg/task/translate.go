package task

import (
	"encoding/json"

	"github.com/mcpchecker/mcpchecker/pkg/steps"
	"github.com/mcpchecker/mcpchecker/pkg/util"
)

func translateV1Alpha1ToSteps(legacy *TaskStepsV1Alpha1) (*TaskSpec, error) {
	var err error
	spec := &TaskSpec{
		Prompt: legacy.Prompt,
	}

	spec.Setup, err = translateLegacyStep(legacy.SetupScript)
	if err != nil {
		return nil, err
	}

	spec.Cleanup, err = translateLegacyStep(legacy.CleanupScript)
	if err != nil {
		return nil, err
	}

	if legacy.VerifyScript != nil {
		if legacy.VerifyScript.Step != nil {
			spec.Verify, err = translateLegacyStep(legacy.VerifyScript.Step)
			if err != nil {
				return nil, err
			}
		} else if legacy.VerifyScript.LLMJudgeStepConfig != nil {
			raw, err := json.Marshal(legacy.VerifyScript.LLMJudgeStepConfig)
			if err != nil {
				return nil, err
			}

			spec.Verify = []steps.StepConfig{{
				"llmJudge": raw,
			}}
		}

	}

	return spec, nil
}

func translateLegacyStep(step *util.Step) ([]steps.StepConfig, error) {
	if step == nil || step.IsEmpty() {
		return []steps.StepConfig{}, nil
	}

	raw, err := json.Marshal(step)
	if err != nil {
		return []steps.StepConfig{}, err
	}

	return []steps.StepConfig{{
		"script": raw,
	}}, nil
}
