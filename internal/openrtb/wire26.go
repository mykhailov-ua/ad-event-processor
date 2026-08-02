package openrtb

type BidRequest struct {
	ID     string   `json:"id"`
	Imp    []Imp    `json:"imp"`
	Site   *Site    `json:"site,omitempty"`
	App    *App     `json:"app,omitempty"`
	DOOH   *DOOH    `json:"dooh,omitempty"`
	Device Device   `json:"device"`
	User   *User    `json:"user,omitempty"`
	Regs   *Regs    `json:"regs,omitempty"`
	Source *Source  `json:"source,omitempty"`
	BCat   []string `json:"bcat,omitempty"`
	BAdv   []string `json:"badv,omitempty"`
	BApp   []string `json:"bapp,omitempty"`
	AT     int      `json:"at,omitempty"`
	Cur    []string `json:"cur,omitempty"`
	Tmax   int      `json:"tmax,omitempty"`
	Test   int      `json:"test,omitempty"`
}

type Imp struct {
	ID          string  `json:"id"`
	Banner      *Banner `json:"banner,omitempty"`
	Video       *Video  `json:"video,omitempty"`
	Audio       *Audio  `json:"audio,omitempty"`
	Native      *Native `json:"native,omitempty"`
	BidFloor    float64 `json:"bidfloor"`
	BidFloorCur string  `json:"bidfloorcur,omitempty"`
	PMP         *PMP    `json:"pmp,omitempty"`
}

type Banner struct {
	W int `json:"w,omitempty"`
	H int `json:"h,omitempty"`
}

type Video struct {
	Mimes       []string `json:"mimes,omitempty"`
	MinDuration int      `json:"minduration,omitempty"`
	MaxDuration int      `json:"maxduration,omitempty"`
	W           int      `json:"w,omitempty"`
	H           int      `json:"h,omitempty"`
}

type Audio struct{}

type Native struct{}

type PMP struct {
	Deals []Deal `json:"deals,omitempty"`
}

type Deal struct {
	ID       string   `json:"id"`
	BidFloor float64  `json:"bidfloor,omitempty"`
	WSeat    []string `json:"wseat,omitempty"`
}

type Site struct {
	Domain string   `json:"domain,omitempty"`
	Page   string   `json:"page,omitempty"`
	Cat    []string `json:"cat,omitempty"`
}

type App struct {
	Bundle string   `json:"bundle,omitempty"`
	Cat    []string `json:"cat,omitempty"`
}

type DOOH struct{}

type Device struct {
	IP         string `json:"ip,omitempty"`
	IPv6       string `json:"ipv6,omitempty"`
	UA         string `json:"ua,omitempty"`
	DeviceType int    `json:"devicetype,omitempty"`
	OS         string `json:"os,omitempty"`
	Geo        *Geo   `json:"geo,omitempty"`
}

type Geo struct {
	Country string `json:"country,omitempty"`
}

type User struct {
	ID string `json:"id,omitempty"`
}

type Regs struct {
	Ext *RegsExt `json:"ext,omitempty"`
}

type RegsExt struct {
	GDPR      *int   `json:"gdpr,omitempty"`
	USPrivacy string `json:"us_privacy,omitempty"`
}

type Source struct {
	Ext *SourceExt `json:"ext,omitempty"`
}

type SourceExt struct {
	Schain *Schain `json:"schain,omitempty"`
}

type Schain struct {
	Complete int          `json:"complete,omitempty"`
	Nodes    []SchainNode `json:"nodes,omitempty"`
}

type SchainNode struct {
	ASI string `json:"asi,omitempty"`
	SID string `json:"sid,omitempty"`
}

type BidResponse struct {
	ID      string    `json:"id"`
	BidID   string    `json:"bidid,omitempty"`
	Cur     string    `json:"cur,omitempty"`
	NBR     int       `json:"nbr,omitempty"`
	SeatBid []SeatBid `json:"seatbid,omitempty"`
}

type SeatBid struct {
	Seat string `json:"seat,omitempty"`
	Bid  []Bid  `json:"bid"`
}

type Bid struct {
	ID      string   `json:"id"`
	ImpID   string   `json:"impid"`
	Price   float64  `json:"price"`
	AdM     string   `json:"adm,omitempty"`
	NURL    string   `json:"nurl,omitempty"`
	BURL    string   `json:"burl,omitempty"`
	AdID    string   `json:"adid,omitempty"`
	CrID    string   `json:"crid,omitempty"`
	CID     string   `json:"cid,omitempty"`
	ADomain []string `json:"adomain,omitempty"`
	Cat     []string `json:"cat,omitempty"`
	DealID  string   `json:"dealid,omitempty"`
}

type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Version string   `json:"version,omitempty"`
	Errors  []string `json:"errors"`
}

type IntegrationProfile struct {
	OpenRTBVersion string   `json:"openrtb_version"`
	Required       []string `json:"required"`
	Supported      []string `json:"supported"`
	NotSupported   []string `json:"not_supported"`
}

type ExchangeConfig struct {
	NoBidMode    string
	MultiImpMax  int
	RegsPolicy   string
	CoppaPolicy  string
	Blocklist    bool
	Delivery     string
	NURLTemplate []byte
	SeatID       []byte
}
