package castle

type Config struct {
	DBLocation  string `help:"database location"`
	Host        string `help:"listening interface"`
	Port        int    `help:"listening port"`
	TLSHostname string `help:"enable https and get a certificate for this hostname"`
	TLSEmail    string `help:"email used to administer certificates on Lets Encrypt"`
	NoUpdates   bool   `help:"disable automatic updates"`
}
