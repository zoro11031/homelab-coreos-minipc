-- Bootstrap lazy.nvim
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    "git", "clone", "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable", lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)

-- Basic settings
vim.g.mapleader = " "
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.mouse = "a"
vim.opt.ignorecase = true
vim.opt.smartcase = true
vim.opt.hlsearch = false
vim.opt.wrap = false
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
vim.opt.expandtab = true
vim.opt.termguicolors = true

-- Plugins
local plugins = {
  -- Color scheme
  {
    "folke/tokyonight.nvim",
    lazy = false,
    priority = 1000,
    config = function()
      vim.cmd.colorscheme("tokyonight-night")
    end,
  },

  -- Treesitter for syntax highlighting
  {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    config = function()
      require("nvim-treesitter.configs").setup({
        ensure_installed = { "lua", "python", "bash", "yaml", "json", "markdown" },
        highlight = { enable = true },
        indent = { enable = true },
      })
    end,
  },

  -- File tree
  {
    "nvim-tree/nvim-tree.lua",
    dependencies = { "nvim-tree/nvim-web-devicons" },
    config = function()
      require("nvim-tree").setup()
    end,
  },

  -- Fuzzy finder
  {
    "nvim-telescope/telescope.nvim",
    dependencies = { "nvim-lua/plenary.nvim" },
  },

  -- LSP and autocomplete
  {
    "neovim/nvim-lspconfig",
    dependencies = {
      "hrsh7th/nvim-cmp",
      "hrsh7th/cmp-nvim-lsp",
      "hrsh7th/cmp-buffer",
      "hrsh7th/cmp-path",
      "L3MON4D3/LuaSnip",
      "saadparwaiz1/cmp_luasnip",
    },
    config = function()
      local cmp = require("cmp")

      cmp.setup({
        snippet = {
          expand = function(args)
            require("luasnip").lsp_expand(args.body)
          end,
        },
        mapping = cmp.mapping.preset.insert({
          ["<C-Space>"] = cmp.mapping.complete(),
          ["<CR>"] = cmp.mapping.confirm({ select = true }),
          ["<Tab>"] = cmp.mapping.select_next_item(),
          ["<S-Tab>"] = cmp.mapping.select_prev_item(),
        }),
        sources = {
          { name = "nvim_lsp" },
          { name = "luasnip" },
          { name = "buffer" },
          { name = "path" },
        },
      })

      local capabilities = require("cmp_nvim_lsp").default_capabilities()
      local lspconfig = require("lspconfig")
      local util = require("lspconfig.util")

      local server_overrides = {
        pyright = {
          root_dir = util.root_pattern("pyproject.toml", "setup.py", "requirements.txt", ".git"),
        },
        lua_ls = {
          settings = {
            Lua = {
              diagnostics = { globals = { "vim" } },
            },
          },
        },
      }

      local servers = { "bashls", "pyright", "yamlls", "lua_ls" }
      for _, server in ipairs(servers) do
        local opts = vim.tbl_deep_extend(
          "force",
          { capabilities = capabilities },
          server_overrides[server] or {}
        )
        lspconfig[server].setup(opts)
      end
    end,
  },

  -- Git signs
  {
    "lewis6991/gitsigns.nvim",
    config = true,
  },

  -- Mini modules (pairs, comment, statusline)
  {
    "echasnovski/mini.nvim",
    config = function()
      require("mini.pairs").setup()
      require("mini.comment").setup()
      require("mini.statusline").setup()
    end,
  },
}

require("lazy").setup(plugins)

-- Keymaps
local function map(mode, lhs, rhs, desc)
  vim.keymap.set(mode, lhs, rhs, { desc = desc })
end

local function nmap(lhs, rhs, desc)
  map("n", lhs, rhs, desc)
end

nmap("<leader>ff", "<cmd>Telescope find_files<cr>", "Find files")
nmap("<leader>fg", "<cmd>Telescope live_grep<cr>", "Live grep")
nmap("<leader>fb", "<cmd>Telescope buffers<cr>", "List buffers")
nmap("<leader>e", "<cmd>NvimTreeToggle<cr>", "Toggle file tree")
