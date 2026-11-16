package health

import "testing"

func TestMifflinStJeorBMR(t *testing.T) {
	bmr, ok := MifflinStJeorBMR(90, 180, 40, "male")
	if !ok || bmr != 1830 {
		t.Fatalf("male got %v ok=%v want 1830", bmr, ok)
	}
	if sedentaryTDEE(bmr) != 2196 {
		t.Fatalf("sedentary %v want 2196", sedentaryTDEE(bmr))
	}
	bmr, ok = MifflinStJeorBMR(70, 165, 35, "female")
	if !ok || bmr != 1395 {
		t.Fatalf("female got %v ok=%v want 1395", bmr, ok)
	}
	if _, ok := MifflinStJeorBMR(90, 180, 40, ""); ok {
		t.Fatal("empty sex should fail")
	}
}
