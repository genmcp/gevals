package task

import (
	"encoding/json"

	"github.com/genmcp/gevals/pkg/steps"
	"github.com/genmcp/gevals/pkg/util"
)

func translateV1Alpha1ToSteps(legacy *TaskStepsV1Alpha1) (*TaskSpec, error) {
	var err error
	spec := &TaskSpec{}

	spec.Setup, err = translateLegacyStep(legacy.SetupScript)
	if err != nil {
		return nil, err
	}

	spec.Cleanup, err = translateLegacyStep(legacy.CleanupScript)
	if err != nil {
		return nil, err
	}

	spec.Verify, err = translateLegacyStep(legacy.VerifyScript.Step)

	if legacy.VerifyScript.LLMJudgeStepConfig != nil {
		raw, err := json.Marshal(legacy.VerifyScript.LLMJudgeStepConfig)
		if err != nil {
			return nil, err
		}

		spec.Verify = append(spec.Verify, steps.StepConfig{
			"llmJudge": raw,
		})
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
