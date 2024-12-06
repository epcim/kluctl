package args

import (
	"fmt"
	"github.com/kluctl/kluctl/lib/yaml"
	"github.com/kluctl/kluctl/v2/pkg/types"
	"strings"
)

type ImageFlags struct {
	FixedImage      []string         `group:"images" short:"F" help:"Pin an image to a given version. Expects '--fixed-image=image<:namespace:deployment:container>=result'"`
	FixedImagesFile ExistingFileType `group:"images" help:"Use .yaml file to pin image versions. See output of list-images sub-command or read the documentation for details about the output format" exts:"yml,yaml"`
}

func (args *ImageFlags) LoadFixedImagesFromArgs() ([]types.FixedImage, error) {
	var ret types.FixedImagesConfig

	if args.FixedImagesFile != "" {
		err := yaml.ReadYamlFile(args.FixedImagesFile.String(), &ret)
		if err != nil {
			return nil, err
		}
	}

	for _, fi := range args.FixedImage {
		e, err := buildFixedImageEntryFromArg(fi)
		if err != nil {
			return nil, err
		}
		ret.Images = append(ret.Images, *e)
	}

	return ret.Images, nil
}

func buildFixedImageEntryFromArg(arg string) (*types.FixedImage, error) {
	s := strings.Split(arg, "=")
	if len(s) != 2 {
		return nil, fmt.Errorf("--fixed-image expects 'image<:namespace:deployment:container>=result'")
	}
	image := s[0]
	result := s[1]

	s = strings.Split(image, ":")
	e := types.FixedImage{
		Image:       &s[0],
		ResultImage: result,
	}

	if len(s) >= 2 {
		e.Namespace = &s[1]
	}
	if len(s) >= 3 {
		e.Deployment = &s[2]
	}
	if len(s) >= 4 {
		e.Container = &s[3]
	}
	if len(s) >= 5 {
		return nil, fmt.Errorf("--fixed-image expects 'image<:namespace:deployment:container>=result'")
	}

	return &e, nil
}
