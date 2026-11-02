package styles

import "github.com/xmhrnx/giode/styles/properties"

// Styles is the set of CSS-like properties applied to an element.
// Zero values mean "unset"; elements fall back to their defaults.
//
// Lengths (sizes, paddings, margins, gaps, radii, border widths,
// offsets, shadows) are in device-independent pixels (dp): they are
// scaled by the display's density when laid out, so a UI keeps its
// physical size on HiDPI screens, matching font sizes, which are in
// sp. Use Px to resolve a Styles value to physical pixels in custom
// elements.
type Styles struct {
	// FlexDirection is the main axis of a flex container: "row" or
	// "column".
	FlexDirection properties.FlexDirection
	// Align positions children on the flex cross axis.
	Align properties.Align
	// Justify distributes children on the flex main axis.
	Justify properties.Justify
	// Wrap controls whether children continue onto a new line when
	// they overflow the main axis. Unset defaults to wrapping on Row
	// containers and not wrapping on Stack (column) containers.
	Wrap properties.Wrap
	// Gap is the spacing between flex children, and between wrapped
	// lines.
	Gap properties.Gap
	// FlexGrow is the share of spare space a flex child takes.
	FlexGrow properties.FlexGrow
	// Width and Height size the element: 0 hugs content, -1 expands,
	// >0 is fixed.
	Width  properties.Width
	Height properties.Height
	// WidthPct and HeightPct size the element as a percentage of the
	// available space and take precedence over Width and Height.
	WidthPct  properties.WidthPct
	HeightPct properties.HeightPct
	// Padding is the space between the edge and the content.
	Padding properties.Padding
	// Margin is the space outside the painted area.
	Margin properties.Margin
	// Background is the fill color behind the content.
	Background properties.Background
	// BackgroundImage is an image behind the content, scaled with
	// BackgroundFit.
	BackgroundImage properties.BackgroundImage
	BackgroundFit   properties.BackgroundFit
	// BorderWidth and BorderColor draw a border around the element.
	BorderWidth properties.BorderWidth
	BorderColor properties.BorderColor
	// BorderRadius rounds the background corners.
	BorderRadius properties.BorderRadius
	// BorderRadii rounds each corner separately and takes precedence
	// over BorderRadius when any corner is set.
	BorderRadii properties.Corners
	// BackgroundGradient paints a linear gradient above Background
	// (see properties.Linear).
	BackgroundGradient properties.BackgroundGradient
	// Opacity is the element transparency in the range 0-1.
	Opacity properties.Opacity
	// Color is the text color.
	Color properties.Color
	// FontSize is the text size in sp.
	FontSize properties.FontSize
	// FontWeight is the text weight.
	FontWeight properties.FontWeight
	// TextAlign aligns text within its box.
	TextAlign properties.TextAlign
	// LineHeight is the distance between text baselines.
	LineHeight properties.LineHeight
	// MaxLines truncates text after the given number of lines.
	MaxLines properties.MaxLines
	// TabWidth is the distance between tab stops in text, in columns
	// (default 4).
	TabWidth properties.TabWidth
	// Cursor is the cursor shown while hovering.
	Cursor properties.Cursor
	// Fit controls how images scale into their box.
	Fit properties.Fit
	// Position selects static, relative or absolute placement.
	Position properties.Position
	// Top, Right, Bottom and Left offset absolutely positioned
	// elements from their parent box edges.
	Top    properties.Top
	Right  properties.Right
	Bottom properties.Bottom
	Left   properties.Left
	// ZIndex orders siblings: higher values paint above.
	ZIndex properties.ZIndex
	// MinWidth, MaxWidth, MinHeight and MaxHeight clamp the resolved
	// size.
	MinWidth  properties.MinWidth
	MaxWidth  properties.MaxWidth
	MinHeight properties.MinHeight
	MaxHeight properties.MaxHeight
	// BoxShadow paints a drop shadow behind the element.
	BoxShadow properties.BoxShadow
	// ScrollBar customizes the scrollbar a Scroll container draws.
	ScrollBar properties.ScrollBar
	// AspectRatio derives the unresized axis from the resized one.
	AspectRatio properties.AspectRatio
	// Overflow clips content to the box when set to hidden.
	Overflow properties.Overflow
	// TextDecoration decorates text.
	TextDecoration properties.TextDecoration
	// FontStyle selects normal or italic text.
	FontStyle properties.FontStyle
	// FontFamily selects the text typeface.
	FontFamily properties.FontFamily
	// TextTransform changes the capitalization of text.
	TextTransform properties.TextTransform
	// TextOverflow selects how text cut off by MaxLines ends.
	TextOverflow properties.TextOverflow
	// Transform rotates, scales or translates the painted element
	// without affecting layout.
	Transform properties.Transform
	// Display removes the element from layout when set to none.
	Display properties.Display
	// Visibility hides the element while keeping its space.
	Visibility properties.Visibility
	// Transition animates color and opacity changes between base and
	// hover styles.
	Transition properties.Transition
}
