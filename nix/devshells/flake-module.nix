{
  perSystem =
    {
      config,
      pkgs,
      ...
    }:
    {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          pkgs.delve
          pkgs.go
          pkgs.golangci-lint
          pkgs.pre-commit
        ];

        _GO_VERSION = "${pkgs.go.version}";

        # Disable hardening for fortify otherwize it's not possible to use Delve.
        hardeningDisable = [ "fortify" ];

        shellHook = ''
          ${config.pre-commit.installationScript}

          if [[ "$(${pkgs.gnugrep}/bin/grep '^\(go \)[0-9.]*$' go.mod)" != "go ''${_GO_VERSION}" ]]; then
            ${pkgs.gnused}/bin/sed -e "s:^\(go \)[0-9.]*$:\1''${_GO_VERSION}:" -i go.mod
          fi
        '';
      };
    };
}
