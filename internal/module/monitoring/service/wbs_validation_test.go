package monitoring_service

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateWBSNodesValidRootAndLeafWeight(t *testing.T) {
	rootA := uuid.New()
	rootB := uuid.New()

	validation := validateWBSNodes([]wbsNode{
		{ID: rootA, Weight: 40},
		{ID: rootB, Weight: 60},
	})

	if !validationIsValid(validation) {
		t.Fatalf("expected valid WBS, got errors: %v", validation.Errors)
	}
	if validation.TotalRootWeight != 100 {
		t.Fatalf("expected total root weight 100, got %d", validation.TotalRootWeight)
	}
	if validation.TotalLeafWeight != 100 {
		t.Fatalf("expected total leaf weight 100, got %d", validation.TotalLeafWeight)
	}
}

func TestValidateWBSNodesDetectsOverweight(t *testing.T) {
	validation := validateWBSNodes([]wbsNode{
		{ID: uuid.New(), Weight: 80},
		{ID: uuid.New(), Weight: 30},
	})

	if !hasValidationError(validation.Errors, "TOTAL_ROOT_WEIGHT_EXCEEDS_100") {
		t.Fatalf("expected TOTAL_ROOT_WEIGHT_EXCEEDS_100, got %v", validation.Errors)
	}
	if !hasValidationError(validation.Errors, "TOTAL_LEAF_WEIGHT_EXCEEDS_100") {
		t.Fatalf("expected TOTAL_LEAF_WEIGHT_EXCEEDS_100, got %v", validation.Errors)
	}
}

func TestValidateWBSNodesDetectsCycle(t *testing.T) {
	taskA := uuid.New()
	taskB := uuid.New()

	validation := validateWBSNodes([]wbsNode{
		{ID: taskA, ParentID: uuid.NullUUID{UUID: taskB, Valid: true}, Weight: 50},
		{ID: taskB, ParentID: uuid.NullUUID{UUID: taskA, Valid: true}, Weight: 50},
	})

	if !hasValidationError(validation.Errors, "WBS_CYCLE_DETECTED") {
		t.Fatalf("expected WBS_CYCLE_DETECTED, got %v", validation.Errors)
	}
}

func TestValidateWBSNodesDetectsEmptySiblingAndOrphanCases(t *testing.T) {
	if validation := validateWBSNodes(nil); !hasValidationError(validation.Errors, "WBS_EMPTY") {
		t.Fatalf("expected WBS_EMPTY, got %v", validation.Errors)
	}

	parent := uuid.New()
	validation := validateWBSNodes([]wbsNode{
		{ID: uuid.New(), ParentID: uuid.NullUUID{UUID: parent, Valid: true}, Weight: 10},
	})
	if !hasValidationError(validation.Errors, "ORPHAN_TASK") {
		t.Fatalf("expected ORPHAN_TASK, got %v", validation.Errors)
	}

	parent = uuid.New()
	validation = validateWBSNodes([]wbsNode{
		{ID: parent, Weight: 100},
		{ID: uuid.New(), ParentID: uuid.NullUUID{UUID: parent, Valid: true}, Weight: 60},
		{ID: uuid.New(), ParentID: uuid.NullUUID{UUID: parent, Valid: true}, Weight: 50},
	})
	if !hasValidationError(validation.Errors, "SIBLING_WEIGHT_EXCEEDS_100") {
		t.Fatalf("expected SIBLING_WEIGHT_EXCEEDS_100, got %v", validation.Errors)
	}
}

func validationIsValid(validation wbsValidation) bool {
	return len(validation.Errors) == 0
}

func hasValidationError(errors []string, expected string) bool {
	for _, err := range errors {
		if err == expected {
			return true
		}
	}
	return false
}
