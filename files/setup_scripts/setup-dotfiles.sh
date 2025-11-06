#!/bin/bash
# Setup dotfiles for the core user
# This script clones the dotfiles repo and sets up zsh configuration

set -euo pipefail

DOTFILES_REPO="https://github.com/zoro11031/dotfiles.git"
DOTFILES_DIR="${HOME}/.dotfiles"
USER_HOME="${HOME}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up dotfiles for ${USER}...${NC}"

# Clone dotfiles repository if it doesn't exist
if [ ! -d "${DOTFILES_DIR}" ]; then
    echo -e "${YELLOW}Cloning dotfiles repository...${NC}"
    git clone "${DOTFILES_REPO}" "${DOTFILES_DIR}"
else
    echo -e "${YELLOW}Dotfiles directory already exists, pulling latest changes...${NC}"
    cd "${DOTFILES_DIR}"
    git pull
fi

cd "${DOTFILES_DIR}"

# Run the install script if it exists
if [ -f "${DOTFILES_DIR}/install.sh" ]; then
    echo -e "${YELLOW}Running dotfiles install script...${NC}"
    bash "${DOTFILES_DIR}/install.sh"
else
    echo -e "${YELLOW}No install.sh found, using GNU Stow directly...${NC}"
    # Stow all directories in the dotfiles repo
    for dir in */; do
        if [ -d "$dir" ]; then
            echo -e "${YELLOW}Stowing ${dir%/}...${NC}"
            stow -v -t "${USER_HOME}" "${dir%/}" || true
        fi
    done
fi

# Change default shell to zsh if not already set
if [ "${SHELL}" != "$(which zsh)" ]; then
    echo -e "${YELLOW}Changing default shell to zsh...${NC}"
    # Note: On CoreOS/immutable systems, we may need to use chsh differently
    # or just set it in the user's profile
    if command -v chsh &> /dev/null; then
        chsh -s "$(which zsh)" || echo -e "${YELLOW}Warning: Could not change shell. You may need to run 'chsh -s \$(which zsh)' manually.${NC}"
    else
        echo -e "${YELLOW}chsh not available. Add 'exec zsh' to your .bash_profile to auto-switch to zsh.${NC}"
    fi
fi

echo -e "${GREEN}Dotfiles setup complete!${NC}"
echo -e "${GREEN}Please log out and log back in for changes to take effect.${NC}"
