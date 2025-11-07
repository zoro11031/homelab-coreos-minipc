if vim.loader then
  vim.loader.enable()
end

-- Leader
vim.g.mapleader = " "
vim.g.maplocalleader = ","

-- Core options
local opt = vim.opt
opt.number = true
opt.relativenumber = true
opt.mouse = "a"
opt.ignorecase = true
opt.smartcase = true
opt.hlsearch = false
opt.wrap = false
opt.tabstop = 4
opt.shiftwidth = 4
opt.expandtab = true
opt.termguicolors = true
opt.signcolumn = "yes"
opt.splitbelow = true
opt.splitright = true

-- Bootstrap mini.nvim for dependency management via mini.deps
local mini_path = vim.fn.stdpath("data") .. "/site/pack/deps/start/mini.nvim"
if not vim.loop.fs_stat(mini_path) then
  vim.fn.system({
    "git",
    "clone",
    "https://github.com/echasnovski/mini.nvim",
    mini_path,
  })
end
vim.opt.rtp:prepend(mini_path)

local MiniDeps = require("mini.deps")
MiniDeps.setup({ path = { package = vim.fn.stdpath("data") .. "/site/pack/deps" } })

local add = MiniDeps.add
add("echasnovski/mini.nvim")
add("lewis6991/gitsigns.nvim")

-- Mini modules
require("mini.pairs").setup()
require("mini.comment").setup()
require("mini.statusline").setup()
require("mini.files").setup()
require("mini.pick").setup()

-- Git signs
require("gitsigns").setup()

-- LSP keymaps on attach
vim.api.nvim_create_autocmd("LspAttach", {
  callback = function(args)
    local opts = { buffer = args.buf }
    vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)
    vim.keymap.set("n", "gr", vim.lsp.buf.references, opts)
    vim.keymap.set("n", "K", vim.lsp.buf.hover, opts)
    vim.keymap.set({ "n", "v" }, "<leader>cf", vim.lsp.buf.format, opts)
    vim.keymap.set("n", "[d", vim.diagnostic.goto_prev, opts)
    vim.keymap.set("n", "]d", vim.diagnostic.goto_next, opts)
    vim.keymap.set("n", "<leader>cd", vim.diagnostic.open_float, opts)
  end,
})

-- Configure LSP servers using vim.lsp.config (nvim 0.11+)
-- Load lspconfig plugin first, then use the new vim.lsp.config API
local lspconfig = vim.lsp.config
for _, server in ipairs({ "bashls", "pyright", "yamlls", "lua_ls" }) do
  lspconfig[server].setup(server == "lua_ls" and {
    settings = {
      Lua = {
        diagnostics = { globals = { "vim" } },
        workspace = { library = vim.api.nvim_get_runtime_file("", true) },
      },
    },
  } or {})
end

-- Keymaps
local pick = require("mini.pick")
vim.keymap.set("n", "<leader>ff", pick.builtin.files, { desc = "Find files" })
vim.keymap.set("n", "<leader>fg", pick.builtin.grep_live, { desc = "Live grep" })
vim.keymap.set("n", "<leader>fb", pick.builtin.buffers, { desc = "List buffers" })
vim.keymap.set("n", "<leader>e", require("mini.files").open, { desc = "Toggle file explorer" })
vim.keymap.set("n", "<leader>qq", vim.cmd.quit, { desc = "Quit" })
vim.keymap.set("n", "<leader>ww", vim.cmd.write, { desc = "Write file" })

