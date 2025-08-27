{
  ci.testers.go.enable = true;
  ci.linters = {
    editorconfig-checker.settings.Exclude = ["./double/internal/*_generated*.go"];
    golangci-lint = {
      enable = true;
      linters = {
        exclusions = {
          paths = ["internal/example"];
          rules = [
            {
              path = "api.go";
              linters = ["gocritic"];
              text = "hugeParam: serverAddress is heavy";
            }
          ];
        };
      };
    };
  };
}
