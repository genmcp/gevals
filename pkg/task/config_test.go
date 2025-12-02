package task

import (
	"fmt"
	"testing"

	"github.com/genmcp/gevals/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	basePath = "testdata"
)

func TestFromFile(t *testing.T) {
	tt := map[string]struct {
		file      string
		expected  *TaskSpec
		expectErr bool
	}{
		"create pod inline": {
			file: "create-pod-inline.yaml",
			expected: &TaskSpec{
				TypeMeta: util.TypeMeta{
					Kind: KindTask,
				},
				Metadata: TaskMetadata{
					Name:       "create pod inline",
					Difficulty: DifficultyEasy,
				},
				Steps: TaskSteps{
					SetupScript: &util.Step{
						Inline: `#!/usr/bin/env bash
kubectl delete namespace create-pod-test --ignore-not-found
kubectl create namespace create-pod-test`,
					},
					VerifyScript: &VerifyStep{
						Step: &util.Step{
							Inline: `#!/usr/bin/env bash
if kubectl wait --for=condition=Ready pod/web-server -n create-pod-test --timeout=120s; then
    exit 0
else
    exit 1
fi`,
						},
					},
					CleanupScript: &util.Step{
						Inline: `#!/usr/bin/env bash
kubectl delete pod web-server -n create-pod-test --ignore-not-found
kubectl delete namespace create-pod-test --ignore-not-found`,
					},
					Prompt: &util.Step{
						Inline: "Please create a nginx pod named web-server in the create-pod-test namespace",
					},
				},
			},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got, err := FromFile(fmt.Sprintf("%s/%s", basePath, tc.file))
			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
