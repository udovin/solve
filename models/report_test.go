package models

import (
	"testing"
)

func TestReportStore_getLocker(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewReportStore(testDB, "test_report", "test_report_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestReportStore_Modify(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewReportStore(testDB, "test_report", "test_report_change")
	report := Report{
		CreateTime: 1,
	}
	if err := store.Create(&report); err != nil {
		t.Fatal(err)
	}
	if report.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, err := store.Get(report.ID)
	if err != nil {
		t.Fatal("Unable to found report")
	}
	if found.CreateTime != report.CreateTime {
		t.Fatal("Report has invalid create time")
	}
	report.CreateTime = 2
	if err := store.Update(&report); err != nil {
		t.Fatal(err)
	}
	found, err = store.Get(report.ID)
	if err != nil {
		t.Fatal("Unable to found report")
	}
	if found.CreateTime != report.CreateTime {
		t.Fatal("Report has invalid create time")
	}
	if err := store.Delete(report.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(report.ID); err == nil {
		t.Fatal("Report should be deleted")
	}
}
