package main

import (
	gittypes "github.com/kluctl/kluctl/lib/git/types"
	"github.com/kluctl/kluctl/v2/pkg/types"
	"github.com/kluctl/kluctl/v2/pkg/types/result"
	"github.com/kluctl/kluctl/v2/pkg/utils/uo"
	"github.com/kluctl/kluctl/v2/pkg/webui"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	converter := typescriptify.New().
		WithBackupDir("").
		Add(result.CommandResult{}).
		Add(result.CommandResultSummary{}).
		Add(result.ValidateResult{}).
		Add(result.ValidateResultSummary{}).
		Add(result.DriftDetectionResult{}).
		Add(result.ChangedObject{}).
		Add(webui.ShortName{}).
		Add(uo.UnstructuredObject{}).
		Add(webui.ProjectTargetKey{}).
		Add(webui.AuthInfo{}).
		ManageType(gittypes.GitUrl{}, typescriptify.TypeOptions{TSType: "string"}).
		ManageType(gittypes.GitRef{}, typescriptify.TypeOptions{TSType: "GitRef", TSTransform: "new GitRef(__VALUE__)"}).
		ManageType(gittypes.RepoKey{}, typescriptify.TypeOptions{TSType: "string"}).
		ManageType(types.YamlUrl{}, typescriptify.TypeOptions{TSType: "string"}).
		ManageType(uo.UnstructuredObject{}, typescriptify.TypeOptions{TSType: "any"}).
		ManageType(metav1.Time{}, typescriptify.TypeOptions{TSType: "string"}).
		ManageType(apiextensionsv1.JSON{}, typescriptify.TypeOptions{TSType: "any"})

	converter.AddImport("import { GitRef } from './models-static'")

	err := converter.ConvertToFile("ui/src/models.ts")
	if err != nil {
		panic(err.Error())
	}
}
