package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
)

func toolInputSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, serverTool := range (&Toolset{}).GetTools(nil) {
		if serverTool.Tool.Name != name {
			continue
		}
		data, err := json.Marshal(serverTool.Tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal %s input schema: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("unmarshal %s input schema: %v", name, err)
		}
		return schema
	}
	t.Fatalf("tool %q not found", name)
	return nil
}

func schemaProperties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("input schema has no properties object")
	}
	return properties
}

func schemaRequires(schema map[string]any, property string) bool {
	required, _ := schema["required"].([]any)
	for _, item := range required {
		if item == property {
			return true
		}
	}
	return false
}

func TestToolset_WriteTools_ExcludesDeprecatedCloseAlerts(t *testing.T) {
	for _, serverTool := range (&Toolset{}).GetTools(nil) {
		if serverTool.Tool.Name == "close_alerts" {
			t.Fatal("close_alerts must not be exposed because the current FlashDuty OpenAPI has no alert close operation")
		}
	}
}

func TestToolset_UsesPublishedListFields(t *testing.T) {
	incidentProperties := schemaProperties(t, toolInputSchema(t, "list_incidents"))
	if _, ok := incidentProperties["query"]; !ok {
		t.Fatal("list_incidents must expose query")
	}
	for _, unsupported := range []string{"title", "labels"} {
		if _, ok := incidentProperties[unsupported]; ok {
			t.Errorf("list_incidents must not expose %q", unsupported)
		}
	}

	alertProperties := schemaProperties(t, toolInputSchema(t, "list_alerts"))
	for _, unsupported := range []string{"title", "labels"} {
		if _, ok := alertProperties[unsupported]; ok {
			t.Errorf("list_alerts must not expose %q", unsupported)
		}
	}
	brief, ok := alertProperties["brief"].(map[string]any)
	if !ok {
		t.Fatalf("list_alerts brief schema = %T, want object", alertProperties["brief"])
	}
	briefDescription, _ := brief["description"].(string)
	if !strings.Contains(briefDescription, "alert_status") {
		t.Errorf("brief description = %q, want it to contain alert_status", briefDescription)
	}
	if strings.Contains(briefDescription, "is_active") {
		t.Errorf("brief description = %q, must not list is_active as a response field", briefDescription)
	}
}

func TestToolset_MatchesPublishedOptionalFields(t *testing.T) {
	for _, check := range []struct {
		tool     string
		property string
	}{
		{tool: "create_incident", property: "title"},
		{tool: "reopen_incidents", property: "reason"},
		{tool: "comment_incidents", property: "comment"},
		{tool: "list_schedules", property: "team_ids"},
	} {
		t.Run(check.tool, func(t *testing.T) {
			schema := toolInputSchema(t, check.tool)
			if _, ok := schemaProperties(t, schema)[check.property]; !ok {
				t.Errorf("%s must expose %s", check.tool, check.property)
			}
			if schemaRequires(schema, check.property) {
				t.Errorf("%s must not require %s", check.tool, check.property)
			}
		})
	}
}

func TestToolset_AssignmentUsesStringEscalationRuleID(t *testing.T) {
	properties := schemaProperties(t, toolInputSchema(t, "assign_incident"))
	rule, ok := properties["escalate_rule_id"].(map[string]any)
	if !ok {
		t.Fatalf("escalate_rule_id schema = %T, want object", properties["escalate_rule_id"])
	}
	if rule["type"] != "string" {
		t.Errorf("escalate_rule_id type = %v, want string", rule["type"])
	}
}

func TestToolset_UsesIntegerSchemaForIntegerInputs(t *testing.T) {
	for _, check := range []struct {
		tool     string
		property string
	}{
		{tool: "get_channel", property: "channel_id"},
		{tool: "list_incidents", property: "start_time"},
		{tool: "list_schedules", property: "limit"},
	} {
		t.Run(check.tool, func(t *testing.T) {
			property, ok := schemaProperties(t, toolInputSchema(t, check.tool))[check.property].(map[string]any)
			if !ok {
				t.Fatalf("%s schema = %T, want object", check.property, schemaProperties(t, toolInputSchema(t, check.tool))[check.property])
			}
			if property["type"] != "integer" {
				t.Errorf("%s type = %v, want integer", check.property, property["type"])
			}
		})
	}

	array, ok := schemaProperties(t, toolInputSchema(t, "list_schedules"))["team_ids"].(map[string]any)
	if !ok {
		t.Fatal("team_ids schema must be an object")
	}
	items, ok := array["items"].(map[string]any)
	if !ok {
		t.Fatal("team_ids items schema must be an object")
	}
	if items["type"] != "integer" {
		t.Errorf("team_ids item type = %v, want integer", items["type"])
	}
}

func TestToolset_IncidentStatsMatchesPublishedLimits(t *testing.T) {
	properties := schemaProperties(t, toolInputSchema(t, "get_incident_stats"))
	for _, check := range []struct {
		property string
		want     string
	}{
		{property: "end_time", want: "one year"},
		{property: "severities", want: "'Ok'"},
	} {
		t.Run(check.property, func(t *testing.T) {
			property, ok := properties[check.property].(map[string]any)
			if !ok {
				t.Fatalf("%s schema = %T, want object", check.property, properties[check.property])
			}
			description, _ := property["description"].(string)
			if !strings.Contains(description, check.want) {
				t.Errorf("%s description = %q, want it to contain %q", check.property, description, check.want)
			}
		})
	}
}
