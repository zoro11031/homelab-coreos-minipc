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
add("neovim/nvim-lspconfig")

-- Mini modules
require("mini.pairs").setup()
require("mini.comment").setup()
require("mini.statusline").setup()
require("mini.files").setup()
require("mini.pick").setup()

-- Git signs
require("gitsigns").setup()

-- LSP
local lspconfig = require("lspconfig")
local capabilities = vim.lsp.protocol.make_client_capabilities()

local function on_attach(client, bufnr)
  local map = function(mode, lhs, rhs, desc)
    vim.keymap.set(mode, lhs, rhs, { buffer = bufnr, desc = desc })
  end

  map("n", "gd", vim.lsp.buf.definition, "LSP: Go to definition")
  map("n", "gr", vim.lsp.buf.references, "LSP: Go to references")
  map("n", "K", vim.lsp.buf.hover, "LSP: Hover")
  map({ "n", "v" }, "<leader>cf", vim.lsp.buf.format, "LSP: Format")
  map("n", "[d", vim.diagnostic.goto_prev, "LSP: Previous diagnostic")
  map("n", "]d", vim.diagnostic.goto_next, "LSP: Next diagnostic")
  map("n", "<leader>cd", vim.diagnostic.open_float, "LSP: Line diagnostics")
end

local servers = { "bashls", "pyright", "yamlls", "lua_ls" }
for _, server in ipairs(servers) do
  local opts = { capabilities = capabilities, on_attach = on_attach }
  if server == "lua_ls" then
    opts.settings = {
      Lua = {
        diagnostics = { globals = { "vim" } },
        workspace = { library = vim.api.nvim_get_runtime_file("", true) },
      },
    }
  end
  lspconfig[server].setup(opts)
end

-- Keymaps
local function map(mode, lhs, rhs, desc)
  vim.keymap.set(mode, lhs, rhs, { desc = desc })
end

local pick = require("mini.pick")
map("n", "<leader>ff", pick.builtin.files, "Find files")
map("n", "<leader>fg", pick.builtin.grep_live, "Live grep")
map("n", "<leader>fb", pick.builtin.buffers, "List buffers")
map("n", "<leader>e", require("mini.files").open, "Toggle file explorer")
map("n", "<leader>qq", vim.cmd.quit, "Quit")
map("n", "<leader>ww", vim.cmd.write, "Write file")

