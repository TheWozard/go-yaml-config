package config

import "fmt"

func ExampleLoad() {
	type AppConfig struct {
		Name string `yaml:"name" env-default:"app"`
	}

	// A missing path is skipped, leaving the env-default in place.
	cfg, err := Load[AppConfig]("does-not-exist.yaml")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.Name)
	// Output: app
}

func ExampleMustLoad() {
	type AppConfig struct {
		Name string `yaml:"name" env-default:"app"`
	}

	cfg := MustLoad[AppConfig]("does-not-exist.yaml")

	fmt.Println(cfg.Name)
	// Output: app
}
