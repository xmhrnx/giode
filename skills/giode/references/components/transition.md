# Transition

Animates a component in and out. It takes the component and three options: the effect, the
speed and the easing.

```go
details := giode.Transition(panel, giode.FadeUp, giode.TransitionNormal, giode.EaseOut)

giode.Button("Details").OnClick(details.Toggle)
```

Create it once (it holds the animation state) and lay it out every frame. A visible
transition animates in on its first layout; `Show`, `Hide` and `Toggle` animate it in and
out, reversing smoothly when interrupted. While hidden it takes no space.

## Effects

| Kind | Effect |
| --- | --- |
| `Fade` | Opacity from 0. |
| `SlideUp`, `SlideDown`, `SlideLeft`, `SlideRight` | Moves in from a short distance away. |
| `Enlarge` | Grows from smaller. |
| `Shrink` | Settles from larger. |
| `Rotate` | Turns in from a quarter turn. |
| `Collapse`, `CollapseWidth` | Animates the space taken in the layout (height or width), pushing neighbors and clipping the content — for expanding panels. |

Combine kinds with `|`: `giode.Fade | giode.SlideLeft`. Ready-made combinations: `FadeUp`
(fade + slide up), `Zoom` (fade + enlarge), `Pop` (fade + shrink). Transforms are
paint-only: apart from `Collapse`, the component keeps its full size in the layout while it
animates, so nothing around it jumps.

## Speed

Any `time.Duration`, or the presets `TransitionFast` (150ms), `TransitionNormal` (250ms) and
`TransitionSlow` (500ms). Zero means normal.

## Easing

CSS timing functions: `EaseOut` (fast start, gentle end — usually right for entering
content), `EaseIn`, `EaseInOut` (the default), `Ease` (CSS's default), `EaseLinear`, plus
`EaseOutBack` (slight overshoot), `EaseInBack` and `EaseOutBounce`. Any CSS curve works too:

```go
giode.Easing("cubic-bezier(0.34, 1.56, 0.64, 1)")
```

The same easings drive `Styles.Transition` for hover effects.

## Methods

| Method | Effect |
| --- | --- |
| `Show()`, `Hide()`, `Toggle()`, `SetVisible(v)` | Animate in or out. |
| `Visible()`, `Animating()`, `Progress()` | Current state. |
| `Hidden()` | Start hidden instead of animating in. |
| `Appear(false)` | Start fully shown, without the entry animation. |
| `OnDone(fn)` | Called with the final visibility when an animation ends. |
| `Kind(k)`, `Duration(d)`, `Easing(e)`, `Child(el)` | Change the options at any time. |

Transitions request frames only while animating, and work inside `Memo`: a running
transition keeps its memo live, and a settled one lets it cache again.
