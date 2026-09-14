package main

import "testing"

func TestBuildAutoStartCommandQuotesExePath(t *testing.T) {
	cases := []struct {
		name string
		exe  string
		want string
	}{
		{name: "普通路径", exe: `C:\Apps\workbuddy-checkin.exe`, want: `"C:\Apps\workbuddy-checkin.exe" --hidden`},
		{name: "含空格路径", exe: `C:\Program Files\workbuddy-checkin.exe`, want: `"C:\Program Files\workbuddy-checkin.exe" --hidden`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAutoStartCommand(tc.exe); got != tc.want {
				t.Fatalf("buildAutoStartCommand(%q) = %q, want %q", tc.exe, got, tc.want)
			}
		})
	}
}

func TestHiddenFromArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "双破折号", args: []string{"--hidden"}, want: true},
		{name: "单破折号", args: []string{"-hidden"}, want: true},
		{name: "显式真值", args: []string{"--hidden=true"}, want: true},
		{name: "显式假值", args: []string{"--hidden=false"}, want: false},
		{name: "无参数", args: []string{}, want: false},
		{name: "无关参数", args: []string{"--other"}, want: false},
		{name: "非 flag 参数在前则停止解析", args: []string{"foo", "--hidden"}, want: false},
		{name: "终止符后不解析", args: []string{"--", "--hidden"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hiddenFromArgs(tc.args); got != tc.want {
				t.Fatalf("hiddenFromArgs(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
