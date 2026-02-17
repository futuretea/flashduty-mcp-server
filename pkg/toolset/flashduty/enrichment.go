package flashduty

import (
	"encoding/json"
	"fmt"
	"sync"
)

// enrichIncidentList enriches a list of incidents with person and channel names.
func enrichIncidentList(c *Client, jsonData string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return jsonData
	}

	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return jsonData
	}

	personIDs := make(map[int]struct{})
	channelIDs := make(map[int]struct{})
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			collectIncidentIDs(obj, personIDs, channelIDs)
		}
	}

	personNames, channelNames := fetchNamesConcurrently(c, personIDs, channelIDs)
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			injectIncidentNames(obj, personNames, channelNames)
		}
	}

	return marshalOrFallback(data, jsonData)
}

// enrichIncidentDetail enriches a single incident detail with person and channel names.
func enrichIncidentDetail(c *Client, jsonData string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return jsonData
	}

	personIDs := make(map[int]struct{})
	channelIDs := make(map[int]struct{})
	collectIncidentIDs(data, personIDs, channelIDs)

	personNames, channelNames := fetchNamesConcurrently(c, personIDs, channelIDs)
	injectIncidentNames(data, personNames, channelNames)

	return marshalOrFallback(data, jsonData)
}

// enrichTimeline enriches timeline items with operator person names.
func enrichTimeline(c *Client, jsonData string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return jsonData
	}

	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return jsonData
	}

	personIDs := make(map[int]struct{})
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			collectIntID(obj, "operator_id", personIDs)
			collectIntID(obj, "person_id", personIDs)
		}
	}

	if len(personIDs) == 0 {
		return jsonData
	}

	personNames, _ := c.FetchPersonInfos(intSetToSlice(personIDs))
	if personNames == nil {
		return jsonData
	}

	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			injectPersonName(obj, "operator_id", "operator_name", personNames)
			injectPersonName(obj, "person_id", "person_name", personNames)
		}
	}

	return marshalOrFallback(data, jsonData)
}

// fetchNamesConcurrently fetches person and channel names in parallel.
func fetchNamesConcurrently(c *Client, personIDSet map[int]struct{}, channelIDSet map[int]struct{}) (map[int]string, map[int]string) {
	var (
		personNames  map[int]string
		channelNames map[int]string
		wg           sync.WaitGroup
	)

	if len(personIDSet) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			personNames, _ = c.FetchPersonInfos(intSetToSlice(personIDSet))
		}()
	}

	if len(channelIDSet) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			channelNames, _ = c.FetchChannelInfos(intSetToSlice(channelIDSet))
		}()
	}

	wg.Wait()

	if personNames == nil {
		personNames = make(map[int]string)
	}
	if channelNames == nil {
		channelNames = make(map[int]string)
	}
	return personNames, channelNames
}

// collectIncidentIDs extracts all person and channel IDs from an incident object.
func collectIncidentIDs(obj map[string]any, personIDs, channelIDs map[int]struct{}) {
	collectIntID(obj, "creator_id", personIDs)
	collectIntID(obj, "closer_id", personIDs)
	collectIntID(obj, "channel_id", channelIDs)
	collectResponderPersonIDs(obj, personIDs)
	collectAssignedToPersonIDs(obj, personIDs)
}

// injectIncidentNames injects resolved person and channel names into an incident object.
func injectIncidentNames(obj map[string]any, personNames, channelNames map[int]string) {
	injectPersonName(obj, "creator_id", "creator_name", personNames)
	injectPersonName(obj, "closer_id", "closer_name", personNames)
	injectChannelName(obj, "channel_id", "channel_name", channelNames)
	enrichResponders(obj, personNames)
	enrichAssignedTo(obj, personNames)
}

// marshalOrFallback marshals data to indented JSON, returning fallback on error.
func marshalOrFallback(data any, fallback string) string {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fallback
	}
	return string(out)
}

// collectIntID adds a non-zero integer ID from obj[field] to the set.
func collectIntID(obj map[string]any, field string, set map[int]struct{}) {
	if id := toInt(obj[field]); id != 0 {
		set[id] = struct{}{}
	}
}

// collectResponderPersonIDs collects person IDs from the responders array.
func collectResponderPersonIDs(obj map[string]any, set map[int]struct{}) {
	responders, ok := obj["responders"].([]any)
	if !ok {
		return
	}
	for _, r := range responders {
		if resp, ok := r.(map[string]any); ok {
			collectIntID(resp, "person_id", set)
		}
	}
}

// collectAssignedToPersonIDs collects person IDs from the assigned_to object.
func collectAssignedToPersonIDs(obj map[string]any, set map[int]struct{}) {
	assignedTo, ok := obj["assigned_to"].(map[string]any)
	if !ok {
		return
	}
	personIDs, ok := assignedTo["person_ids"].([]any)
	if !ok {
		return
	}
	for _, pid := range personIDs {
		if id := toInt(pid); id != 0 {
			set[id] = struct{}{}
		}
	}
}

// injectPersonName injects a resolved name field next to the ID field.
func injectPersonName(obj map[string]any, idField, nameField string, names map[int]string) {
	if id := toInt(obj[idField]); id != 0 {
		if name, ok := names[id]; ok {
			obj[nameField] = name
		}
	}
}

// injectChannelName injects a resolved channel name if not already present.
func injectChannelName(obj map[string]any, idField, nameField string, names map[int]string) {
	if existing, ok := obj[nameField].(string); ok && existing != "" {
		return
	}
	if id := toInt(obj[idField]); id != 0 {
		if name, ok := names[id]; ok {
			obj[nameField] = name
		}
	}
}

// enrichResponders enriches each responder with person_name.
func enrichResponders(obj map[string]any, personNames map[int]string) {
	responders, ok := obj["responders"].([]any)
	if !ok {
		return
	}
	for _, r := range responders {
		if resp, ok := r.(map[string]any); ok {
			injectPersonName(resp, "person_id", "person_name", personNames)
		}
	}
}

// enrichAssignedTo enriches person_ids in assigned_to with a person_names array.
func enrichAssignedTo(obj map[string]any, personNames map[int]string) {
	assignedTo, ok := obj["assigned_to"].(map[string]any)
	if !ok {
		return
	}
	personIDs, ok := assignedTo["person_ids"].([]any)
	if !ok {
		return
	}
	names := make([]string, 0, len(personIDs))
	for _, pid := range personIDs {
		id := toInt(pid)
		if name, ok := personNames[id]; ok {
			names = append(names, name)
		} else {
			names = append(names, fmt.Sprintf("person_%d", id))
		}
	}
	assignedTo["person_names"] = names
}

// intSetToSlice converts a set of ints to a slice.
func intSetToSlice(set map[int]struct{}) []int {
	result := make([]int, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	return result
}
