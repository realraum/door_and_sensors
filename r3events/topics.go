// (c) Bernhard Tittelbach, 2015

package r3events

const (
	TOPIC_R3                       string = "realraum/"
	TOPIC_ACTIONS                  string = "action/"
	CLIENTID_FRONTDOOR             string = "frontdoor"
	CLIENTID_BACKDOOR              string = "backdoorcx"
	CLIENTID_PILLAR                string = "pillar"
	CLIENTID_OLGAFREEZER           string = "olgafreezer"
	CLIENTID_META                  string = "metaevt"
	CLIENTID_GW                    string = "gateway"
	CLIENTID_LASERCUTTER           string = "lasercutter"
	CLIENTID_XMPPBOT               string = "xmppbot"
	CLIENTID_XBEE                  string = "xbee"
	CLIENTID_IRCBOT                string = "ircchanbot"
	CLIENTID_LIGHTCTRL             string = "GoLightCtrl"
	CLIENTID_WEBFRONT              string = "GoMQTTWebFront"
	CLIENTID_PIPELEDS              string = "PipeLEDs"
	CLIENTID_CEILING1              string = "ceiling1"
	CLIENTID_CEILING2              string = "ceiling2"
	CLIENTID_CEILING3              string = "ceiling3"
	CLIENTID_CEILING4              string = "ceiling4"
	CLIENTID_CEILING5              string = "ceiling5"
	CLIENTID_CEILING6              string = "ceiling6"
	CLIENTID_CEILING7              string = "ceiling7"
	CLIENTID_CEILING8              string = "ceiling8"
	CLIENTID_CEILINGALL            string = "ceilingAll"
	CLIENTID_BASICLIGHT1              string = "basiclight1"
	CLIENTID_BASICLIGHT2              string = "basiclight2"
	CLIENTID_BASICLIGHT3              string = "basiclight3"
	CLIENTID_BASICLIGHT4              string = "basiclight4"
	CLIENTID_BASICLIGHT5              string = "basiclight5"
	CLIENTID_BASICLIGHT6              string = "basiclight6"
	CLIENTID_BASICLIGHT7              string = "basiclight7"
	CLIENTID_BASICLIGHT8              string = "basiclight8"
	CLIENTID_BASICLIGHTALL            string = "basiclightAll"
	ACTDESTID_RF433                string = "rf433"
	ACTDESTID_YAMAHA               string = "yamahastereo"
	TYPE_LOCK                      string = "lock"
	TYPE_AJAR                      string = "ajar"
	TYPE_CMDEVT                    string = "cmdevt"
	TYPE_PROBLEM                   string = "problemevt"
	TYPE_MANUALLOCK                string = "manuallockmovement"
	TYPE_DOOMBUTTON                string = "boredoombuttonpressed"
	TYPE_TEMP                      string = "temperature"
	TYPE_ILLUMINATION              string = "illumination"
	TYPE_DUST                      string = "dust"
	TYPE_RELHUMIDITY               string = "relhumidity"
	TYPE_MOVEMENTPIR               string = "movement"
	TYPE_TEMPOVER                  string = "overtemp"
	TYPE_SENSORLOST                string = "sensorlost"
	TYPE_GASALERT                  string = "gasalert"
	TYPE_POWERLOSS                 string = "powerloss"
	TYPE_DUSKORDAWN                string = "duskordawn"
	TYPE_VOLTAGE                   string = "voltage"
	TYPE_LIGHT                     string = "light"
	TYPE_DEFAULTLIGHT              string = "defaultlight"
	TYPE_PLEASEREPEAT              string = "pleaserepeat"
	TOPIC_FRONTDOOR_LOCK           string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/" + TYPE_LOCK
	TOPIC_FRONTDOOR_AJAR           string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/" + TYPE_AJAR
	TOPIC_FRONTDOOR_CMDEVT         string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/" + TYPE_CMDEVT
	TOPIC_FRONTDOOR_PROBLEM        string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/" + TYPE_PROBLEM
	TOPIC_FRONTDOOR_MANUALLOCK     string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/" + TYPE_MANUALLOCK
	TOPIC_FRONTDOOR_RAWFWLINES     string = TOPIC_R3 + CLIENTID_FRONTDOOR + "/rawfwlines"
	TOPIC_PILLAR_DOOMBUTTON        string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_DOOMBUTTON
	TOPIC_PILLAR_TEMP              string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_TEMP
	TOPIC_PILLAR_ILLUMINATION      string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_ILLUMINATION
	TOPIC_PILLAR_DUST              string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_DUST
	TOPIC_PILLAR_RELHUMIDITY       string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_RELHUMIDITY
	TOPIC_PILLAR_MOVEMENTPIR       string = TOPIC_R3 + CLIENTID_PILLAR + "/" + TYPE_MOVEMENTPIR
	TOPIC_XBEE_TEMP                string = TOPIC_R3 + CLIENTID_XBEE + "/" + TYPE_TEMP
	TOPIC_XBEE_RELHUMIDITY         string = TOPIC_R3 + CLIENTID_XBEE + "/" + TYPE_RELHUMIDITY
	TOPIC_XBEE_VOLTAGE             string = TOPIC_R3 + CLIENTID_XBEE + "/" + TYPE_VOLTAGE
	TOPIC_BACKDOOR_MOVEMENTPIR     string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_MOVEMENTPIR
	TOPIC_OLGAFREEZER_TEMP         string = TOPIC_R3 + CLIENTID_OLGAFREEZER + "/" + TYPE_TEMP
	TOPIC_OLGAFREEZER_TEMPOVER     string = TOPIC_R3 + CLIENTID_OLGAFREEZER + "/" + TYPE_TEMPOVER
	TOPIC_OLGAFREEZER_SENSORLOST   string = TOPIC_R3 + CLIENTID_OLGAFREEZER + "/" + TYPE_SENSORLOST
	TOPIC_BACKDOOR_TEMP            string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_TEMP
	TOPIC_BACKDOOR_LOCK            string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_LOCK
	TOPIC_BACKDOOR_AJAR            string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_AJAR
	TOPIC_BACKDOOR_GASALERT        string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_GASALERT
	TOPIC_BACKDOOR_POWERLOSS       string = TOPIC_R3 + CLIENTID_BACKDOOR + "/" + TYPE_POWERLOSS
	TOPIC_META_PRESENCE            string = TOPIC_R3 + CLIENTID_META + "/presence"
	TOPIC_META_SENSORLOST          string = TOPIC_R3 + CLIENTID_META + "/" + TYPE_SENSORLOST
	TOPIC_META_REALMOVE            string = TOPIC_R3 + CLIENTID_META + "/" + TYPE_MOVEMENTPIR
	TOPIC_META_TEMPSPIKE           string = TOPIC_R3 + CLIENTID_META + "/TempSensorSpike"
	TOPIC_META_DUSTSPIKE           string = TOPIC_R3 + CLIENTID_META + "/DustSensorSpike"
	TOPIC_META_HUMIDITYSPIKE       string = TOPIC_R3 + CLIENTID_META + "/HumiditySensorSpike"
	TOPIC_META_DUSKORDAWN          string = TOPIC_R3 + CLIENTID_META + "/" + TYPE_DUSKORDAWN
	TOPIC_GW_DHCPACK               string = TOPIC_R3 + CLIENTID_GW + "/NetDHCPACK"
	TOPIC_GW_STATS                 string = TOPIC_R3 + CLIENTID_GW + "/NetGWStatUpdate"
	TOPIC_LASER_CARD               string = TOPIC_R3 + CLIENTID_LASERCUTTER + "/cardpresent"
	TOPIC_IRCBOT_FOODREQUEST       string = TOPIC_R3 + CLIENTID_IRCBOT + "/foodorderrequest"
	TOPIC_IRCBOT_FOODINVITE        string = TOPIC_R3 + CLIENTID_IRCBOT + "/foodorderinvite"
	TOPIC_IRCBOT_FOODETA           string = TOPIC_R3 + CLIENTID_IRCBOT + "/foodordereta"
	ACT_RF433_SEND                 string = TOPIC_ACTIONS + ACTDESTID_RF433 + "/sendcode3byte"
	ACT_RF433_SETDELAY             string = TOPIC_ACTIONS + ACTDESTID_RF433 + "/setdelay"
	ACT_YAMAHA_SEND                string = TOPIC_ACTIONS + ACTDESTID_YAMAHA + "/ircmd"
	ACT_PIPELEDS_RESTART           string = TOPIC_ACTIONS + CLIENTID_PIPELEDS + "/restart"
	ACT_PIPELEDS_PATTERN           string = TOPIC_ACTIONS + CLIENTID_PIPELEDS + "/pattern"
	ACT_LIGHTCTRL_NAME             string = TOPIC_ACTIONS + CLIENTID_LIGHTCTRL + "/name"
	ACT_ALLFANCYLIGHT_PLEASEREPEAT string = TOPIC_ACTIONS + CLIENTID_CEILINGALL + "/" + TYPE_PLEASEREPEAT
)
