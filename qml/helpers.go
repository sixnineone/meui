package qml

import (
	"strings"

	"github.com/sixnineone/meui/layout"
	"github.com/sixnineone/meui/widgets"
)

func toFloat32(v interface{}) (float32, bool) {
	if f, ok := v.(float64); ok {
		return float32(f), true
	}
	if f, ok := v.(int); ok {
		return float32(f), true
	}
	if f, ok := v.(int64); ok {
		return float32(f), true
	}
	return 0, false
}

func parseAlignment(s string) int {
	switch s {
	case "Qt.AlignLeft":
		return layout.AlignLeftV
	case "Qt.AlignRight":
		return layout.AlignRightV
	case "Qt.AlignHCenter":
		return layout.AlignHCenterV
	case "Qt.AlignTop":
		return layout.AlignTopV
	case "Qt.AlignBottom":
		return layout.AlignBottomV
	case "Qt.AlignVCenter":
		return layout.AlignVCenterV
	case "Qt.AlignCenter":
		return layout.AlignCenterV
	default:
		return layout.AlignNoneV
	}
}

func parseAnchorsProp(propName string, propValue interface{}, anchors *layout.Anchors, widget interface{}) *layout.Anchors {
	if anchors == nil {
		anchors = layout.NewAnchors()
	}

	switch propName {
	case "anchors.fill":
		if s, ok := propValue.(string); ok && s == "parent" {
			anchors.Fill = widget
		}
	case "anchors.left":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.Left = &layout.AnchorLine{Target: widget, Line: layout.AnchorLeft}
		}
	case "anchors.right":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.Right = &layout.AnchorLine{Target: widget, Line: layout.AnchorRight}
		}
	case "anchors.top":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.Top = &layout.AnchorLine{Target: widget, Line: layout.AnchorTop}
		}
	case "anchors.bottom":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.Bottom = &layout.AnchorLine{Target: widget, Line: layout.AnchorBottom}
		}
	case "anchors.horizontalCenter":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.HorizontalCenter = &layout.AnchorLine{Target: widget, Line: layout.AnchorHCenter}
		}
	case "anchors.verticalCenter":
		if s, ok := propValue.(string); ok && strings.HasPrefix(s, "parent") {
			anchors.VerticalCenter = &layout.AnchorLine{Target: widget, Line: layout.AnchorVCenter}
		}
	case "anchors.centerIn":
		if s, ok := propValue.(string); ok && s == "parent" {
			anchors.CenterIn = widget
		}
	case "anchors.margins":
		if f, ok := toFloat32(propValue); ok {
			anchors.Margins = f
		}
	case "anchors.leftMargin":
		if f, ok := toFloat32(propValue); ok {
			anchors.LeftMargin = f
		}
	case "anchors.rightMargin":
		if f, ok := toFloat32(propValue); ok {
			anchors.RightMargin = f
		}
	case "anchors.topMargin":
		if f, ok := toFloat32(propValue); ok {
			anchors.TopMargin = f
		}
	case "anchors.bottomMargin":
		if f, ok := toFloat32(propValue); ok {
			anchors.BottomMargin = f
		}
	case "anchors.horizontalCenterOffset":
		if f, ok := toFloat32(propValue); ok {
			anchors.HCenterOffset = f
		}
	case "anchors.verticalCenterOffset":
		if f, ok := toFloat32(propValue); ok {
			anchors.VCenterOffset = f
		}
	}
	return anchors
}

func buildProgressBar(obj *Object) *widgets.ProgressBar {
	pb := widgets.NewProgressBar()
	pb.SetWidth(200)
	pb.SetHeight(24)

	for _, prop := range obj.Properties {
		switch prop.Name {
		case "value":
			if f, ok := prop.Value.(float64); ok {
				pb.SetValue(float32(f))
			}
		case "indeterminate":
			if b, ok := prop.Value.(bool); ok {
				pb.SetIndeterminate(b)
			}
		case "minimum", "min":
			if f, ok := prop.Value.(float64); ok {
				pb.Minimum = float32(f)
			}
		case "maximum", "max":
			if f, ok := prop.Value.(float64); ok {
				pb.Maximum = float32(f)
			}
		case "width":
			if f, ok := prop.Value.(float64); ok {
				pb.SetWidth(float32(f))
			}
		case "height":
			if f, ok := prop.Value.(float64); ok {
				pb.SetHeight(float32(f))
			}
		case "enabled":
			if b, ok := prop.Value.(bool); ok {
				pb.SetEnabled(b)
			}
		}
	}

	return pb
}

func buildProgressBarWithBindings(obj *Object, ctx *BindingContext, registry *BoundWidgetRegistry) *widgets.ProgressBar {
	pb := buildProgressBar(obj)
	if obj.ID != "" {
		registry.Register(obj.ID, pb)
		ctx.RegisterObject(obj)
	}
	processBindings(obj, ctx, pb)
	return pb
}
