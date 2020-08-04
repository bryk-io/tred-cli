#!/bin/sh

# Configure git to access private Go modules and install CI bot credentials
credentials() {
  mkdir -p 0600 "$HOME"/.ssh
  git config --global url."ssh://git@github.com:".insteadOf "https://github.com"
  git config --global url."ssh://git@bitbucket.org:".insteadOf "https://bitbucket.org"
  ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts
  ssh-keyscan -t rsa bitbucket.org >> ~/.ssh/known_hosts

  # The original file must has been previously encrypted:
  # gpg --cipher-algo AES256 -c FILE (interactive)
  # gpg --batch --yes --passphrase="$GPG_PASSPHRASE" --cipher-algo AES256 -c FILE (not interactive)
  #
  # To get a good random passphrase:
  # openssl rand 64 -base64 | tr -d '\n'
  gpg --quiet --batch --yes --decrypt \
  --passphrase="$GPG_PASSPHRASE" \
  --output "$HOME"/.ssh/id_ed25519 \
  ./.github/workflows/assets/deploy_key
  chmod 400 "$HOME"/.ssh/id_ed25519
}

case $1 in
  "credentials")
  credentials
  ;;

  *)
  echo "Invalid target: $1"
  ;;
esac
