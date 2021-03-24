package http

import "testing"

func TestCanonicalHref(t *testing.T) {
	type args struct {
		href string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "/a///a",
			args: args{
				href: "/a///a",
			},
			want: "/a/a",
		},
		{
			name: "//a/a",
			args: args{
				href: "/a///a",
			},
			want: "/a/a",
		},
		{
			name: "/a/a//",
			args: args{
				href: "/a///a",
			},
			want: "/a/a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanonicalHref(tt.args.href); got != tt.want {
				t.Errorf("CanonicalHref() = %v, want %v", got, tt.want)
			}
		})
	}
}
