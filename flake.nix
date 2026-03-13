{
  description = "worktree-manager & devtree – git worktree manager and dev environment orchestrator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    let
      # Overlay that any consumer flake can apply to get devtree/wtm in their pkgs.
      overlay = final: prev: {
        devtree = self.packages.${final.system}.devtree;
        wtm = self.packages.${final.system}.wtm;
      };
    in
    {
      overlays.default = overlay;
    }
    //
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # Common build inputs for Go on Darwin.
        darwinDeps = pkgs.lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
          pkgs.apple-sdk_15
          pkgs.darwin.libresolv
        ];

        # Shared vendor hash – both binaries share the same go.mod.
        # After `git add`-ing all files, run: nix build .#devtree
        # Nix will error with "got: sha256-XXXX…" – paste that hash here.
        goVendorHash = pkgs.lib.fakeHash;

        # Build the devtree binary.
        devtree = pkgs.buildGoModule {
          pname = "devtree";
          version = "0.1.0";
          src = ./.;
          subPackages = [ "cmd/devtree" ];
          vendorHash = goVendorHash;
          buildInputs = darwinDeps;
          meta = {
            description = "Dev environment orchestrator – daemon, DNS, reverse proxy, process management";
            mainProgram = "devtree";
          };
        };

        # Build the wtm binary.
        wtm = pkgs.buildGoModule {
          pname = "wtm";
          version = "0.1.0";
          src = ./.;
          subPackages = [ "cmd/wtm" ];
          vendorHash = goVendorHash;
          buildInputs = darwinDeps;
          meta = {
            description = "Git worktree manager";
            mainProgram = "wtm";
          };
        };

        # Helper to create a simple script package.
        mkTask = name: text:
          pkgs.writeShellScriptBin name ''
            set -euo pipefail
            ${text}
          '';

        tasks = {
          wtm-build = mkTask "wtm-build" ''
            echo "Building wtm..."
            go build -v -o wtm ./cmd/wtm
            echo "Building devtree..."
            go build -v -o devtree ./cmd/devtree
          '';

          wtm-test = mkTask "wtm-test" ''
            go test -v -coverprofile=coverage.out -covermode=atomic ./...
          '';

          wtm-coverage = mkTask "wtm-coverage" ''
            wtm-test
            go tool cover -func=coverage.out
          '';

          wtm-clean = mkTask "wtm-clean" ''
            rm -f wtm devtree coverage.out
          '';

          wtm-install = mkTask "wtm-install" ''
            wtm-build
            mkdir -p ~/.local/bin
            cp wtm ~/.local/bin/wtm
            chmod +x ~/.local/bin/wtm
            mkdir -p ~/.local/share/gh-wtm
            ln -sf ~/.local/bin/wtm ~/.local/share/gh-wtm/gh-wtm
            (cd ~/.local/share/gh-wtm && gh extension install .) || true
            echo '✓ Installed as gh extension: gh wtm'
            echo '✓ Installed standalone: wtm (ensure ~/.local/bin is in PATH)'
            cp devtree ~/.local/bin/devtree
            chmod +x ~/.local/bin/devtree
            mkdir -p ~/.local/share/devtree
            echo '✓ Installed standalone: devtree (ensure ~/.local/bin is in PATH)'
          '';

          wtm-uninstall = mkTask "wtm-uninstall" ''
            gh extension remove gh-wtm || echo 'gh extension not installed'
            rm -rf ~/.local/share/gh-wtm
            rm -f ~/.local/bin/wtm
            echo '✓ Uninstalled gh extension (if it was installed)'
            echo '✓ Removed standalone wtm command'
          '';

          wtm-lint = mkTask "wtm-lint" ''
            golangci-lint run ./...
          '';

          wtm-fmt = mkTask "wtm-fmt" ''
            go fmt ./...
          '';

          wtm-vet = mkTask "wtm-vet" ''
            go vet ./...
          '';

          wtm-tidy = mkTask "wtm-tidy" ''
            go mod tidy
          '';

          wtm-check = mkTask "wtm-check" ''
            wtm-fmt
            wtm-vet
            wtm-lint
            wtm-test
          '';

          wtm-dev = mkTask "wtm-dev" ''
            go build -o wtm ./cmd/wtm
            ./wtm --help
          '';

          wtm-tasks = mkTask "wtm-tasks" ''
            echo "Available tasks:"
            echo "  wtm-build      Build the wtm and devtree binaries"
            echo "  wtm-test       Run unit tests with coverage"
            echo "  wtm-coverage   View test coverage report"
            echo "  wtm-clean      Clean build artifacts"
            echo "  wtm-install    Install as gh extension and standalone command"
            echo "  wtm-uninstall  Uninstall gh extension and standalone command"
            echo "  wtm-lint       Run golangci-lint"
            echo "  wtm-fmt        Format Go code"
            echo "  wtm-vet        Run go vet"
            echo "  wtm-tidy       Tidy Go modules"
            echo "  wtm-check      Run all checks (fmt, vet, lint, test)"
            echo "  wtm-dev        Build and run wtm --help"
            echo "  wtm-tasks      Show this help"
          '';
        };
      in
      {
        # ── Packages ──────────────────────────────────────────────
        packages = {
          inherit devtree wtm;
          default = devtree;
        };

        # ── Dev shell (unchanged) ────────────────────────────────
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go_1_25
            pkgs.golangci-lint
            pkgs.gh
          ]
          ++ builtins.attrValues tasks
          ++ darwinDeps;
        };
      });
}
