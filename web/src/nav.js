// The docs' navigation: sections of pages, each a Markdown file under
// ../docs (path without .md). Pages not listed still open by URL.
export const NAV = [
	{
		title: "Start here",
		items: [
			{ path: "index", label: "Overview" },
			{ path: "getting-started", label: "Getting started" },
			{ path: "recipes", label: "Recipes" },
			{ path: "faq", label: "FAQ" },
		],
	},
	{
		title: "The CLI",
		items: [
			{ path: "cli/index", label: "Install and overview" },
			{ path: "cli/commands", label: "Commands" },
			{ path: "cli/config", label: "giode.toml" },
			{ path: "cli/distribution", label: "Distribution" },
		],
	},
	{
		title: "Guides",
		items: [
			{ path: "elements", label: "Elements" },
			{ path: "styles", label: "Style properties" },
			{ path: "state", label: "State" },
			{ path: "routing", label: "Routing" },
			{ path: "window", label: "Window" },
			{ path: "platform", label: "Platform services" },
			{ path: "support", label: "Reports and donations" },
			{ path: "mobile", label: "Mobile" },
			{ path: "threading", label: "Threading" },
			{ path: "performance", label: "Rendering performance" },
			{ path: "mcp", label: "MCP server" },
			{ path: "extending", label: "Extending the library" },
			{ path: "low-level", label: "Low-level access" },
			{ path: "architecture", label: "Architecture" },
		],
	},
	{
		title: "Components",
		items: [
			{ path: "components", label: "Overview" },
			{ path: "components/button", label: "Button" },
			{ path: "components/input", label: "Input" },
			{ path: "components/controls", label: "Switch, Segmented, Chip, Badge, Avatar" },
			{ path: "components/tabs", label: "Tabs" },
			{ path: "components/scroll", label: "Scroll" },
			{ path: "components/lazylist", label: "LazyList" },
			{ path: "components/virtuallist", label: "VirtualList" },
			{ path: "components/treeview", label: "TreeView" },
			{ path: "components/table", label: "Table" },
			{ path: "components/drawer", label: "Drawer" },
			{ path: "components/dialog", label: "Dialog" },
			{ path: "components/sheet", label: "Sheet" },
			{ path: "components/menu", label: "Menu" },
			{ path: "components/tooltip", label: "Tooltip" },
			{ path: "components/toaster", label: "Toaster" },
			{ path: "components/pressable", label: "Pressable" },
			{ path: "components/swipeable", label: "Swipeable" },
			{ path: "components/accordion", label: "Accordion" },
			{ path: "components/pagination", label: "Pagination" },
			{ path: "components/appshell", label: "AppShell" },
			{ path: "components/titlebar", label: "Title bar" },
			{ path: "components/transition", label: "Transition" },
			{ path: "components/progress", label: "Progress" },
			{ path: "components/spinner", label: "Spinner" },
			{ path: "components/image", label: "Image" },
			{ path: "components/icons", label: "Icons" },
			{ path: "components/title", label: "Title" },
		],
	},
	{
		title: "Project",
		items: [
			{ path: "contributing", label: "Contributing" },
		],
	},
];

export const PAGES = NAV.flatMap((s) => s.items.map((it) => ({ ...it, section: s.title })));
