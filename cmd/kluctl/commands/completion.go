package commands

import (
	"context"
	"github.com/kluctl/kluctl/v2/cmd/kluctl/args"
	"github.com/kluctl/kluctl/v2/pkg/kluctl_project"
	"github.com/kluctl/kluctl/v2/pkg/status"
	"github.com/kluctl/kluctl/v2/pkg/utils"
	"github.com/spf13/cobra"
	"reflect"
	"strings"
	"sync"
	"time"
)

func RegisterFlagCompletionFuncs(cmdStruct interface{}, ccmd *cobra.Command) error {
	v := reflect.ValueOf(cmdStruct).Elem()
	projectFlags := v.FieldByName("ProjectFlags")
	argsFlags := v.FieldByName("ArgsFlags")
	targetFlags := v.FieldByName("TargetFlags")
	inclusionFlags := v.FieldByName("InclusionFlags")
	imageFlags := v.FieldByName("ImageFlags")

	ctx := context.Background()

	if projectFlags.IsValid() && targetFlags.IsValid() {
		var argsFlag2 *args.ArgsFlags
		if argsFlags.IsValid() {
			argsFlag2 = argsFlags.Addr().Interface().(*args.ArgsFlags)
		}
		_ = ccmd.RegisterFlagCompletionFunc("target", buildTargetCompletionFunc(ctx, projectFlags.Addr().Interface().(*args.ProjectFlags), argsFlag2))
	}

	if projectFlags.IsValid() && inclusionFlags.IsValid() {
		tagsFunc := buildInclusionCompletionFunc(ctx, cmdStruct, false)
		dirsFunc := buildInclusionCompletionFunc(ctx, cmdStruct, true)
		_ = ccmd.RegisterFlagCompletionFunc("include-tag", tagsFunc)
		_ = ccmd.RegisterFlagCompletionFunc("exclude-tag", tagsFunc)
		_ = ccmd.RegisterFlagCompletionFunc("include-deployment-dir", dirsFunc)
		_ = ccmd.RegisterFlagCompletionFunc("exclude-deployment-dir", dirsFunc)
	}

	if imageFlags.IsValid() {
		_ = ccmd.RegisterFlagCompletionFunc("fixed-image", buildImagesCompletionFunc(ctx, cmdStruct))
	}

	return nil
}

func withProjectForCompletion(ctx context.Context, projectArgs *args.ProjectFlags, argsFlags *args.ArgsFlags, cb func(ctx context.Context, p *kluctl_project.LoadedKluctlProject) error) error {
	// let's not update git caches too often
	projectArgs.GitCacheUpdateInterval = time.Second * 60
	return withKluctlProjectFromArgs(ctx, *projectArgs, argsFlags, nil, nil, false, false, true, func(ctx context.Context, p *kluctl_project.LoadedKluctlProject) error {
		return cb(ctx, p)
	})
}

func buildTargetCompletionFunc(ctx context.Context, projectArgs *args.ProjectFlags, argsFlags *args.ArgsFlags) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var ret []string
		err := withProjectForCompletion(ctx, projectArgs, argsFlags, func(ctx context.Context, p *kluctl_project.LoadedKluctlProject) error {
			for _, t := range p.Targets {
				ret = append(ret, t.Name)
			}
			return nil
		})
		if err != nil {
			status.Error(ctx, err.Error())
			return nil, cobra.ShellCompDirectiveError
		}
		return ret, cobra.ShellCompDirectiveDefault
	}
}

func buildAutocompleteProjectTargetCommandArgs(cmdStruct interface{}) projectTargetCommandArgs {
	ptArgs := projectTargetCommandArgs{}

	cmdV := reflect.ValueOf(cmdStruct).Elem()
	if cmdV.FieldByName("ProjectFlags").IsValid() {
		ptArgs.projectFlags = cmdV.FieldByName("ProjectFlags").Interface().(args.ProjectFlags)
	}
	if cmdV.FieldByName("TargetFlags").IsValid() {
		ptArgs.targetFlags = cmdV.FieldByName("TargetFlags").Interface().(args.TargetFlags)
	}
	if cmdV.FieldByName("ArgsFlags").IsValid() {
		ptArgs.argsFlags = cmdV.FieldByName("ArgsFlags").Interface().(args.ArgsFlags)
	}
	if cmdV.FieldByName("ImageFlags").IsValid() {
		ptArgs.imageFlags = cmdV.FieldByName("ImageFlags").Interface().(args.ImageFlags)
	}
	if cmdV.FieldByName("InclusionFlags").IsValid() {
		ptArgs.inclusionFlags = cmdV.FieldByName("InclusionFlags").Interface().(args.InclusionFlags)
	}

	ptArgs.forCompletion = true
	return ptArgs
}

func buildInclusionCompletionFunc(ctx context.Context, cmdStruct interface{}, forDirs bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ptArgs := buildAutocompleteProjectTargetCommandArgs(cmdStruct)

		var tags utils.OrderedMap[bool]
		var deploymentItemDirs utils.OrderedMap[bool]
		var mutex sync.Mutex

		err := withProjectForCompletion(ctx, &ptArgs.projectFlags, &ptArgs.argsFlags, func(ctx context.Context, p *kluctl_project.LoadedKluctlProject) error {
			var targets []string
			if ptArgs.targetFlags.Target == "" {
				for _, t := range p.Targets {
					targets = append(targets, t.Name)
				}
			} else {
				targets = append(targets, ptArgs.targetFlags.Target)
			}

			var wg sync.WaitGroup
			for _, t := range targets {
				ptArgs := ptArgs
				ptArgs.targetFlags.Target = t
				wg.Add(1)
				go func() {
					_ = withProjectTargetCommandContext(ctx, ptArgs, p, func(cmdCtx *commandCtx) error {
						mutex.Lock()
						defer mutex.Unlock()
						for _, di := range cmdCtx.targetCtx.DeploymentCollection.Deployments {
							tags.Merge(di.Tags)
							deploymentItemDirs.Set(di.RelToSourceItemDir, true)
						}
						return nil
					})
					wg.Done()
				}()
			}
			wg.Wait()
			return nil
		})
		if err != nil {
			status.Error(ctx, err.Error())
			return nil, cobra.ShellCompDirectiveError
		}
		if forDirs {
			return deploymentItemDirs.ListKeys(), cobra.ShellCompDirectiveDefault
		} else {
			return tags.ListKeys(), cobra.ShellCompDirectiveDefault
		}
	}
}

func buildImagesCompletionFunc(ctx context.Context, cmdStruct interface{}) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ptArgs := buildAutocompleteProjectTargetCommandArgs(cmdStruct)

		if strings.Index(toComplete, "=") != -1 {
			return nil, cobra.ShellCompDirectiveDefault
		}

		var images utils.OrderedMap[bool]
		var mutex sync.Mutex

		err := withProjectForCompletion(ctx, &ptArgs.projectFlags, &ptArgs.argsFlags, func(ctx context.Context, p *kluctl_project.LoadedKluctlProject) error {
			var targets []string
			if ptArgs.targetFlags.Target == "" {
				for _, t := range p.Targets {
					targets = append(targets, t.Name)
				}
			} else {
				targets = append(targets, ptArgs.targetFlags.Target)
			}

			var wg sync.WaitGroup
			for _, t := range targets {
				ptArgs := ptArgs
				ptArgs.targetFlags.Target = t
				wg.Add(1)
				go func() {
					_ = withProjectTargetCommandContext(ctx, ptArgs, p, func(cmdCtx *commandCtx) error {
						err := cmdCtx.targetCtx.DeploymentCollection.Prepare()
						if err != nil {
							status.Error(ctx, err.Error())
						}

						mutex.Lock()
						defer mutex.Unlock()
						for _, si := range cmdCtx.images.SeenImages(false) {
							str := *si.Image
							if si.Namespace != nil {
								str += ":" + *si.Namespace
							}
							if si.Deployment != nil {
								str += ":" + *si.Deployment
							}
							if si.Container != nil {
								str += ":" + *si.Container
							}
							images.Set(str, true)
						}
						return nil
					})
					wg.Done()
				}()
			}
			wg.Wait()
			return nil
		})
		if err != nil {
			status.Error(ctx, err.Error())
			return nil, cobra.ShellCompDirectiveError
		}
		return images.ListKeys(), cobra.ShellCompDirectiveNoSpace
	}
}
