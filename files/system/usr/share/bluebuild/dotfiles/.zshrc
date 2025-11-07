# Enable Powerlevel10k instant prompt. Should stay close to the top of ~/.zshrc.
# Initialization code that may require console input (password prompts, [y/n]
# confirmations, etc.) must go above this block; everything else may go below.
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# Ensure a usable terminfo entry before plugins depend on it.
if ! infocmp "${TERM:-}" >/dev/null 2>&1; then
  fallback_term="xterm-256color"
  previous_term="${TERM:-unknown}"
  export TERM="$fallback_term"
  # Do not set COLORTERM; let applications detect color capabilities via TERM.
  printf 'zsh: warning: terminal "%s" not recognized; defaulting to %s\n' \
    "$previous_term" "$fallback_term" >&2
fi
unset -v fallback_term previous_term

if [[ -f "/opt/homebrew/bin/brew" ]]; then
  # If you're using macOS, you'll want this enabled
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Set the directory we want to store zinit and plugins
ZINIT_HOME="${XDG_DATA_HOME:-${HOME}/.local/share}/zinit/zinit.git"

# Download Zinit, if it's not there yet
if [ ! -d "$ZINIT_HOME" ]; then
   mkdir -p "$(dirname $ZINIT_HOME)"
   git clone https://github.com/zdharma-continuum/zinit.git "$ZINIT_HOME"
fi

# Source/Load zinit
source "${ZINIT_HOME}/zinit.zsh"

# Create completions directory if it doesn't exist
# This prevents symlink errors when zinit installs completion files
mkdir -p "${XDG_DATA_HOME:-${HOME}/.local/share}/zinit/completions"

# Add in Powerlevel10k
zinit ice depth=1; zinit light romkatv/powerlevel10k

# Determine if fzf can be used safely in the current terminal
fzf_ready=0
if command -v fzf >/dev/null && infocmp "$TERM" >/dev/null 2>&1; then
  fzf_ready=1
fi

# Add in zsh plugins
# NOTE: fzf-tab is NOT loaded here - it must be loaded AFTER compinit
zinit_plugins=(
  zsh-users/zsh-completions
  zsh-users/zsh-autosuggestions
  zdharma-continuum/fast-syntax-highlighting
)

for plugin in "${zinit_plugins[@]}"; do
  zinit light "$plugin"
done


# Add in snippets
# Keep the baseline OMZ snippets here; add additional ones below when required.
zinit snippet OMZL::git.zsh
zinit snippet OMZP::git
zinit snippet OMZP::sudo

# ============================================================================
# Completion System Setup
# ============================================================================

# Set completion styles BEFORE initializing the completion system
# Case-insensitive completion matching
zstyle ':completion:*' matcher-list 'm:{a-z}={A-Za-z}'
# Use LS_COLORS for file completion coloring
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"
# Disable default menu - fzf-tab will provide a better UI
zstyle ':completion:*' menu no

# fzf-tab specific previews (only used if fzf-tab is loaded)
zstyle ':fzf-tab:complete:cd:*' fzf-preview 'ls --color $realpath'
zstyle ':fzf-tab:complete:__zoxide_z:*' fzf-preview 'ls --color $realpath'

# Initialize the completion system
autoload -Uz compinit && compinit

# Load fzf-tab AFTER compinit
# fzf-tab wraps the completion system, so it must be loaded after compinit initializes it
if (( fzf_ready )); then
  zinit light Aloxaf/fzf-tab
fi

# Replay compdefs for any plugins that were loaded before compinit
zinit cdreplay -q

# Explicitly bind Tab to completion if fzf-tab is NOT loaded
# When fzf-tab is loaded, it handles the Tab binding itself
if (( ! fzf_ready )); then
  bindkey '^I' expand-or-complete
fi

# To customize prompt, run `p10k configure` or edit ~/.p10k.zsh.
[[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh

# Keybindings
bindkey -e
bindkey '^p' history-search-backward
bindkey '^n' history-search-forward
bindkey '^[w' kill-region

# Fix Delete, Home, End keys
bindkey '^[[3~' delete-char                    # Delete key
bindkey '^[[H' beginning-of-line               # Home key
bindkey '^[[F' end-of-line                     # End key
bindkey '^[[1~' beginning-of-line              # Home key (alternate)
bindkey '^[[4~' end-of-line                    # End key (alternate)

# Use terminfo for better terminal compatibility (if available)
if (( ${+terminfo} )); then
  [[ -n "${terminfo[kdch1]}" ]] && bindkey "${terminfo[kdch1]}" delete-char
  [[ -n "${terminfo[khome]}" ]] && bindkey "${terminfo[khome]}" beginning-of-line
  [[ -n "${terminfo[kend]}" ]] && bindkey "${terminfo[kend]}" end-of-line
fi

# History
HISTSIZE=5000
HISTFILE=~/.zsh_history
SAVEHIST=$HISTSIZE
HISTDUP=erase
setopt appendhistory
setopt sharehistory
setopt hist_ignore_space
setopt hist_ignore_all_dups
setopt hist_save_no_dups
setopt hist_ignore_dups
setopt hist_find_no_dups

# Dotfiles location
export DOTFILES="${DOTFILES:-$HOME/.dotfiles}"

# Aliases
alias ls='ls --color'
alias vim='nvim'
alias c='clear'
alias restow='(cd "$DOTFILES" && ./install.sh -r)'

# Shell integrations
if command -v fzf >/dev/null; then
  eval "$(fzf --zsh)"
fi

if command -v zoxide >/dev/null; then
  eval "$(zoxide init --cmd cd zsh)"
fi

alias please='sudo $(fc -ln -1)'
alias zshrc='${=EDITOR} ${ZDOTDIR:-$HOME}/.zshrc'

# Grep with color
alias grep='grep --color'
alias sgrep='grep -R -n -H -C 5 --exclude-dir={.git,.svn,CVS}'

# History shortcuts
alias h='history'
alias hgrep="fc -El 0 | grep"

# Disk usage
alias dud='du -d 1 -h'
(( $+commands[duf] )) || alias duf='du -sh *'

# Directory navigation shortcuts
alias ...='../..'
alias ....='../../..'
alias .....='../../../..'
alias d='dirs -v | head -10'

# ========================================
# Global Aliases (for piping)
# ========================================
alias -g H='| head'
alias -g T='| tail'
alias -g G='| grep'
alias -g L="| less"
alias -g NE="2> /dev/null"
alias -g NUL="> /dev/null 2>&1"