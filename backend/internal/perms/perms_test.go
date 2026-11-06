package perms

import "testing"

func TestAdminPassesEverything(t *testing.T) {
	if !Mask(Admin).Has(ManageUsers, ManageRequests) || !Mask(Admin).HasAny(ManageSettings) {
		t.Fatal("admin must pass every check")
	}
	if Mask(Request).Has(ManageRequests) || Mask(Request).HasAny(ManageRequests, ManageUsers) {
		t.Fatal("a plain flag must not pass for another")
	}
	if !Mask(Request|ManageUsers).Has(Request, ManageUsers) || Mask(Request).Has(Request, ManageUsers) {
		t.Fatal("Has needs every flag")
	}
}

func TestNormalizeAddsImpliedFlagsAndDropsUnknownBits(t *testing.T) {
	m := Normalize(ManageRequests)
	for _, f := range []Mask{AutoApprove, AutoApproveMovie, AutoApproveTV, RequestAdvanced, RequestView} {
		if m&f == 0 {
			t.Errorf("manage requests must bring flag %d", f)
		}
	}
	if m := Normalize(Request); m&RequestMovie == 0 || m&RequestTV == 0 {
		t.Error("request must bring movies and shows")
	}
	if m := Normalize(ManageIssues); m&ViewIssues == 0 || m&CreateIssues == 0 {
		t.Error("manage issues must bring view and create")
	}
	// bit 1 (none) and 64 (vote in Seerr) are not ours
	if m := Normalize(1 | 64 | Request); m&(1|64) != 0 {
		t.Errorf("unknown bits survived: %d", m)
	}
	if Normalize(Default) != Default {
		t.Error("the default mask must already be consistent")
	}
}

func TestEffectiveAndRequestRules(t *testing.T) {
	if Effective(true, None) != All {
		t.Error("an admin gets every flag")
	}
	if Effective(false, ManageRequests)&AutoApproveTV == 0 {
		t.Error("effective mask is normalised")
	}
	plain := Default
	if plain.AutoApproves("movie", false) || plain.AutoApproves("tv", false) {
		t.Error("the default must not auto-approve")
	}
	if !(Request|AutoApproveMovie).AutoApproves("movie", false) || (Request|AutoApproveMovie).AutoApproves("tv", false) {
		t.Error("auto-approve per type")
	}
	if !Mask(ManageRequests).AutoApproves("tv", true) {
		t.Error("managers are auto-approved")
	}
	if !(RequestMovie).MayRequest("movie") || (RequestMovie).MayRequest("tv") || None.MayRequest("movie") {
		t.Error("request per type")
	}
}

func TestStoredNeverCarriesTheAdminBit(t *testing.T) {
	if Stored(Admin|Request)&Admin != 0 {
		t.Fatal("the admin bit must not be stored")
	}
	if Effective(false, Admin)&Admin != 0 {
		t.Fatal("a stored admin bit must not make someone admin")
	}
	if Stored(ManageRequests)&AutoApprove == 0 {
		t.Fatal("stored masks are normalised")
	}
}

func TestCatalogIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	var bits Mask
	for _, f := range Catalog {
		if seen[f.Name] {
			t.Errorf("duplicate flag name %s", f.Name)
		}
		seen[f.Name] = true
		if f.Value&(f.Value-1) != 0 {
			t.Errorf("%s is not a single bit", f.Name)
		}
		if bits&f.Value != 0 {
			t.Errorf("%s reuses a bit", f.Name)
		}
		bits |= f.Value
		for _, name := range f.Implies {
			if _, ok := ByName(name); !ok {
				t.Errorf("%s implies unknown flag %s", f.Name, name)
			}
		}
	}
}

func Test4KRights(t *testing.T) {
	if !Default.MayRequest4K("movie") || !Default.MayRequest4K("tv") {
		t.Error("the default keeps 4K")
	}
	if m := Normalize(Request4K); m&Request4KMovie == 0 || m&Request4KTV == 0 {
		t.Error("request 4K must bring movies and shows")
	}
	if m := Normalize(ManageRequests); m&(Request4K|AutoApprove4KMovie|AutoApprove4KTV) != Request4K|AutoApprove4KMovie|AutoApprove4KTV {
		t.Error("manage requests must bring the 4K rights")
	}
	if (Request | Request4KMovie).MayRequest4K("tv") || !(Request | Request4KMovie).MayRequest4K("movie") || Mask(Request).MayRequest4K("movie") {
		t.Error("4K per type")
	}
	if !Mask(Admin).MayRequest4K("tv") {
		t.Error("admins request 4K")
	}
	auto := Mask(Request | AutoApprove)
	if !auto.AutoApproves("movie", false) || auto.AutoApproves("movie", true) {
		t.Error("auto-approve does not cover 4K")
	}
	if m := Mask(Request | AutoApprove4KTV); !m.AutoApproves("tv", true) || m.AutoApproves("movie", true) || m.AutoApproves("tv", false) {
		t.Error("4K auto-approve per type, and only for 4K")
	}
	if !Mask(AutoApprove4K).AutoApproves("movie", true) {
		t.Error("4K auto-approve for both")
	}
}
