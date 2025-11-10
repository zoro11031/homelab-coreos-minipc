--
-- ███╗   ██╗███████╗ ██████╗ ██╗   ██╗██╗███╗   ███╗
-- ████╗  ██║██╔════╝██╔═══██╗██║   ██║██║████╗ ████║
-- ██╔██╗ ██║█████╗  ██║   ██║██║   ██║██║██╔████╔██║
-- ██║╚██╗██║██╔══╝  ██║   ██║╚██╗ ██╔╝██║██║╚██╔╝██║
-- ██║ ╚████║███████╗╚██████╔╝ ╚████╔╝ ██║██║ ╚═╝ ██║
-- ╚═╝  ╚═══╝╚══════╝ ╚═════╝   ╚═══╝  ╚═╝╚═╝     ╚═╝
--
-- Simplified Neovim Config - No Compilation Required
-- Works on barebones systems with just Neovim installed

-- Check Neovim version
if vim.fn.has("nvim-0.8") == 0 then
    error("Need Neovim 0.8+ to use this config")
end

vim.opt.termguicolors = true
vim.deprecate = function() end -- Disable deprecation warnings

-- ============================================================================
-- SETTINGS
-- ============================================================================

local opt = vim.opt
local g = vim.g

g.mapleader = " "

-- File handling
opt.fileencoding = "utf-8"
opt.encoding = "utf-8"
opt.backup = false
opt.swapfile = false
opt.writebackup = false
opt.undofile = true
opt.undodir = vim.fn.stdpath("data") .. "/undo"

-- Indentation
opt.autoindent = true
opt.smartindent = true
opt.expandtab = true
opt.shiftwidth = 4
opt.tabstop = 4
opt.softtabstop = 4
opt.shiftround = true

-- Search
opt.hlsearch = true
opt.ignorecase = true
opt.smartcase = true

-- UI
opt.number = true
opt.cursorline = true
opt.signcolumn = "yes"
opt.scrolloff = 8
opt.sidescrolloff = 8
opt.splitbelow = true
opt.splitright = true
opt.wrap = true
opt.showmode = false
opt.cmdheight = 1
opt.laststatus = 2
opt.list = true
opt.listchars = { tab = "┊ ", trail = "·", extends = "»", precedes = "«" }

-- Performance
opt.updatetime = 100
opt.timeoutlen = 300
opt.history = 100

-- Completion
opt.completeopt = { "menu", "menuone", "noselect" }

-- ============================================================================
-- KEYMAPS
-- ============================================================================

local map = vim.keymap.set

-- Quick save/quit
map("n", "<leader>s", ":w<CR>", { desc = "Save file" })
map("n", "<leader>q", ":qa!<CR>", { desc = "Quit all" })

-- Window navigation
map("n", "<C-h>", "<C-w>h", { desc = "Go to left window" })
map("n", "<C-j>", "<C-w>j", { desc = "Go to lower window" })
map("n", "<C-k>", "<C-w>k", { desc = "Go to upper window" })
map("n", "<C-l>", "<C-w>l", { desc = "Go to right window" })

-- Better indenting
map("v", "<", "<gv")
map("v", ">", ">gv")

-- Move lines
map("n", "<A-j>", ":m .+1<CR>==", { desc = "Move line down" })
map("n", "<A-k>", ":m .-2<CR>==", { desc = "Move line up" })
map("v", "<A-j>", ":m '>+1<CR>gv=gv", { desc = "Move selection down" })
map("v", "<A-k>", ":m '<-2<CR>gv=gv", { desc = "Move selection up" })

-- Clear search highlight
map("n", "<Esc>", ":noh<CR>", { silent = true })

-- Buffer navigation
map("n", "<S-h>", ":bprevious<CR>", { desc = "Previous buffer" })
map("n", "<S-l>", ":bnext<CR>", { desc = "Next buffer" })
map("n", "<leader>x", ":bdelete<CR>", { desc = "Close buffer" })

-- File explorer (netrw)
map("n", "<leader>e", ":Explore<CR>", { desc = "File explorer" })
map("n", "<leader>E", ":Sexplore<CR>", { desc = "Split explorer" })

-- Terminal
map("n", "<leader>t", function()
    local height = math.floor(vim.o.lines / 2)
    vim.cmd("belowright split | resize " .. height .. " | terminal")
end, { desc = "Open terminal" })
map("t", "<Esc>", "<C-\\><C-n>", { desc = "Exit terminal mode" })

-- Better search
map("n", "n", "nzzzv")
map("n", "N", "Nzzzv")

-- Paste without yanking in visual mode
map("x", "<leader>p", '"_dP', { desc = "Paste without yank" })

-- ============================================================================
-- AUTOCOMMANDS
-- ============================================================================

local autocmd = vim.api.nvim_create_autocmd

-- Highlight on yank
autocmd("TextYankPost", {
    callback = function()
        vim.highlight.on_yank({ higroup = "IncSearch", timeout = 200 })
    end,
})

-- Remove trailing whitespace on save
autocmd("BufWritePre", {
    pattern = "*",
    command = ":%s/\\s\\+$//e",
})

-- Disable auto-commenting new lines
autocmd("BufEnter", {
    pattern = "*",
    command = "set fo-=c fo-=r fo-=o",
})

-- File-specific settings
autocmd("FileType", {
    pattern = { "xml", "html", "xhtml", "css", "scss", "javascript", "typescript", "yaml", "lua", "json" },
    command = "setlocal shiftwidth=2 tabstop=2",
})

autocmd("FileType", {
    pattern = { "gitcommit", "markdown", "text" },
    callback = function()
        vim.opt_local.wrap = true
        vim.opt_local.spell = true
    end,
})

-- ============================================================================
-- BOOTSTRAP LAZY.NVIM
-- ============================================================================

local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
    vim.fn.system({
        "git",
        "clone",
        "--filter=blob:none",
        "https://github.com/folke/lazy.nvim.git",
        "--branch=stable",
        lazypath,
    })
end
vim.opt.rtp:prepend(lazypath)

-- ============================================================================
-- PLUGINS (NO COMPILATION REQUIRED)
-- ============================================================================

require("lazy").setup({
    -- Colorscheme - Pure Lua
    {
        "rose-pine/neovim",
        name = "rose-pine",
        priority = 1000,
        config = function()
            require("rose-pine").setup({
                dark_variant = "main",
                disable_italics = true,
            })
            vim.cmd("colorscheme rose-pine")
        end,
    },

    -- File navigation - Pure Lua, no compilation
    {
        "nvim-telescope/telescope.nvim",
        dependencies = { "nvim-lua/plenary.nvim" },
        cmd = "Telescope",
        keys = {
            { "<leader>ff", "<cmd>Telescope find_files<cr>", desc = "Find files" },
            { "<leader>fg", "<cmd>Telescope live_grep<cr>", desc = "Live grep" },
            { "<leader>fb", "<cmd>Telescope buffers<cr>", desc = "Buffers" },
            { "<leader>fh", "<cmd>Telescope help_tags<cr>", desc = "Help" },
            { "<leader>fo", "<cmd>Telescope oldfiles<cr>", desc = "Recent files" },
        },
        config = function()
            require("telescope").setup({
                defaults = {
                    layout_strategy = "horizontal",
                    layout_config = { prompt_position = "top" },
                    sorting_strategy = "ascending",
                    mappings = {
                        i = {
                            ["<C-j>"] = "move_selection_next",
                            ["<C-k>"] = "move_selection_previous",
                        },
                    },
                },
            })
        end,
    },

    -- Mini.files - Pure Lua file explorer
    {
        "echasnovski/mini.files",
        version = false,
        keys = {
            {
                "<leader>fm",
                function()
                    require("mini.files").open(vim.api.nvim_buf_get_name(0), true)
                end,
                desc = "Open mini.files (current file)",
            },
            {
                "<leader>fM",
                function()
                    require("mini.files").open(vim.uv.cwd(), true)
                end,
                desc = "Open mini.files (cwd)",
            },
        },
        config = function()
            require("mini.files").setup({
                windows = { preview = true, width_preview = 50 },
            })
        end,
    },

    -- Comment - Pure Lua
    {
        "numToStr/Comment.nvim",
        keys = { "gc", "gb" },
        config = function()
            require("Comment").setup()
        end,
    },

    -- Autopairs - Pure Lua
    {
        "windwp/nvim-autopairs",
        event = "InsertEnter",
        config = function()
            require("nvim-autopairs").setup({})
        end,
    },

    -- Git signs - Pure Lua
    {
        "lewis6991/gitsigns.nvim",
        event = { "BufReadPre", "BufNewFile" },
        config = function()
            require("gitsigns").setup({
                signs = {
                    add = { text = "│" },
                    change = { text = "│" },
                    delete = { text = "_" },
                    topdelete = { text = "‾" },
                    changedelete = { text = "~" },
                },
            })
        end,
    },

    -- Statusline - Pure Lua
    {
        "nvim-lualine/lualine.nvim",
        event = "VeryLazy",
        config = function()
            require("lualine").setup({
                options = {
                    theme = "auto",
                    component_separators = { left = "|", right = "|" },
                    section_separators = { left = "", right = "" },
                },
                sections = {
                    lualine_a = { "mode" },
                    lualine_b = { "branch", "diff", "diagnostics" },
                    lualine_c = { { "filename", path = 1 } },
                    lualine_x = { "encoding", "fileformat", "filetype" },
                    lualine_y = { "progress" },
                    lualine_z = { "location" },
                },
            })
        end,
    },

    -- Which-key - Pure Lua
    {
        "folke/which-key.nvim",
        event = "VeryLazy",
        config = function()
            require("which-key").setup({})
        end,
    },

    -- LSP Config - Pure Lua (optional, enable if needed)
    {
        "neovim/nvim-lspconfig",
        event = { "BufReadPre", "BufNewFile" },
        config = function()
            -- Basic LSP setup - add your language servers here
            local lspconfig = require("lspconfig")

            -- Example: Lua LSP
            if vim.fn.executable("lua-language-server") == 1 then
                lspconfig.lua_ls.setup({
                    settings = {
                        Lua = {
                            diagnostics = { globals = { "vim" } },
                            workspace = { library = vim.api.nvim_get_runtime_file("", true) },
                        },
                    },
                })
            end

            -- Key mappings for LSP
            vim.api.nvim_create_autocmd("LspAttach", {
                callback = function(ev)
                    local opts = { buffer = ev.buf }
                    vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)
                    vim.keymap.set("n", "K", vim.lsp.buf.hover, opts)
                    vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, opts)
                    vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, opts)
                end,
            })
        end,
    },
}, {
    install = {
        colorscheme = { "rose-pine" },
    },
    checker = {
        enabled = true,
        notify = false,
    },
    change_detection = {
        notify = false,
    },
})

-- ============================================================================
-- NETRW CONFIGURATION (Built-in file explorer)
-- ============================================================================

g.netrw_banner = 0
g.netrw_liststyle = 3
g.netrw_browse_split = 0
g.netrw_altv = 1
g.netrw_winsize = 25

vim.notify("Neovim config loaded successfully!", vim.log.levels.INFO)