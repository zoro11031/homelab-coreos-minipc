# Enable Powerlevel10k instant prompt. Should stay close to the top of ~/.zshrc.
# Initialization code that may require console input (password prompts, [y/n]
# confirmations, etc.) must go above this block; everything else may go below.
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

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

# Add in Powerlevel10k
zinit ice depth=1; zinit light romkatv/powerlevel10k

# Add in zsh plugins
zinit_plugins=(
  zsh-users/zsh-completions
  zsh-users/zsh-autosuggestions
  zdharma-continuum/fast-syntax-highlighting
  Aloxaf/fzf-tab
)

for plugin in "${zinit_plugins[@]}"; do
  zinit light "$plugin"
done


# Add in snippets
# Keep the baseline OMZ snippets here; add additional ones below when required.
zinit snippet OMZL::git.zsh
zinit snippet OMZP::git
zinit snippet OMZP::sudo

# Load completions
autoload -Uz compinit && compinit

zinit cdreplay -q

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

# Completion styling
zstyle ':completion:*' matcher-list 'm:{a-z}={A-Za-z}'
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"
zstyle ':completion:*' menu no
zstyle ':fzf-tab:complete:cd:*' fzf-preview 'ls --color $realpath'
zstyle ':fzf-tab:complete:__zoxide_z:*' fzf-preview 'ls --color $realpath'

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