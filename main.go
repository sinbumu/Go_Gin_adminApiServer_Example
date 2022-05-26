package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aliforever/gista"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/now"
	"github.com/xlzd/gotp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/analytics/v3"
	ga "google.golang.org/api/analyticsreporting/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	prisma "gotchu.admin.server/prisma/generated/prisma-client"
	// prisma "dev.gotchu.admin.server/prisma/generated/prisma-client" //for dev...
)

var (
	pClient         *prisma.Client
	ctx             context.Context
	garService      *ga.Service
	gaKeyPath       string
	firebaseKeyPath string
	firebaseApp     *firebase.App
	awsSession      *session.Session
)

const (
	sessionLifeTime    = 7200000000000 * 3 // 2ㅂㅐㄹㅗ ㅇㅕㄴㅈㅏㅇ
	viewID             = "179724165"
	gaDefaultStartDate = "365daysAgo"
	gaDefaultEndDate   = "today"
	gaVPDataPageSize   = 744
	ninetyDays         = 90 * 24 * time.Hour
	PROJECTNAME        = ""
	slackHook          = "" //live
	slackHookNoti      = "" //live noti
	// slackHook = "" //dev
	// slackHookNoti = "" //dev noti (그냥 후크나 노티 후크나 데브는 똑같음.)
	gwcServerURL = "" //live
	// gwcServerURL = "" //dev
	// gwcServerURL = "" //test
	stibeeSubscribeURL  = "" //현재는 주소록이 픽스고 상황에 따라 변동될 수 있음.
	stibeeAccessToken   = ""
	youtubeChannelURL   = ""
	youtubeAPIKey       = ""
	instagramURL        = "https://www.instagram.com"
	twitchUserInfoURL   = "https://api.twitch.tv/helix/users"
	twitchUserFollowURL = "https://api.twitch.tv/helix/users/follows"
	afreecatvAPIURL     = "http://bjapi.afreecatv.com/api/"

	AWS_ACCESS_KEY_ID      = ""
	AWS_ACCESS_KEY         = ""
	AWS_BUCKET_NAME        = ""
	AWS_BUCKET_FOLDER_NAME = ""
)

type Login struct {
	Email    string `json:"email"`
	Password string `json:"pass"`
}
type GenerateOTP struct {
	Email       string `json:"email"`
	Password    string `json:"pass"`
	UserName    string `json:"userName"`
	AccountName string `json:"accountName"`
}
type OTPCheck struct {
	Value       string `json:"value"`       //otp generate pass code
	AccessEmail string `json:"accessEmail"` //adminUser email
}
type GotchuCashList struct {
	First   int    `json:"first"`
	Skip    int    `json:"skip"`
	MaxDate string `json:"maxDate"`
	MinDate string `json:"minDate"`
}
type GotchuSNSAcountLinkList struct {
	First             int    `json:"first"`
	Skip              int    `json:"skip"`
	OrderBy           string `json:"orderBy"`
	ContainedNickname string `json:"containedNickname"`
	ContainedPageID   string `json:"containedPageId"`
	MaxDate           string `json:"maxDate"`
	MinDate           string `json:"minDate"`
}
type GotchuSNSAcountLinkProcessing struct {
	SNSType         int    `json:"snsType"`         //0 instagram, 1 youtube, 2 afreecaTV, 3 twitch
	UniqueValueType int    `json:"uniqueValueType"` // 0 default (0,2,3 is userId . 1 channelId), 1 (1 userName)
	UniqueValue     string `json:"uniqueValue"`
	PageID          string `json:"pageID"`
	ID              string `json:"id"` //SNSAccountLinkingRequest id
	InstagramCookie string `json:"instagramCookie"`
}
type GotchuReviewList struct {
	First             int    `json:"first"`
	Skip              int    `json:"skip"`
	OrderBy           string `json:"orderBy"`
	ContainedNickname string `json:"containedNickname"`
	MaxDate           string `json:"maxDate"`
	MinDate           string `json:"minDate"`
}
type GotchuTicketUserList struct {
	First             int    `json:"first"`
	Skip              int    `json:"skip"`
	OrderBy           string `json:"orderBy"`
	ContainedNickname string `json:"containedNickname"`
}
type GotchuDeleteReview struct {
	ID string `json:"id"` //reviewContentReviews id
}
type GotchuTicketUserDetail struct {
	PageID string `json:"pageId"` //ticketUserDetail pageId
}
type GotchuPageShopOrderConfirm struct {
	PageShopOrderID string `json:"pageShopOrderId"` //PageShopOrder ID
	UploadImageURL  string `json:"uploadImageURL"`  //인증 이미지 주소
	VideoURL        string `json:"videoUrl"`        //인증 비디오 주소
	Memo            string `json:"memo"`            //인증 글
	Option          string `json:"option"`          //[“ci”,”cv”,”ct”,”ui","uv","ut","di","dv","dt"] 이렇게 9개 옵션 순서대로 생성+이미지 , 생성 + 비디오 , 생성 + 텍스트 …. 이런식
}
type GotchuUpdateUseShopM struct {
	PageID string `json:"pageId"` //reviewContentReviews id
}
type GotchuWebBanner struct {
	BannerNameO string `json:"BannerNameO"` //원래 배너이름, 없으면 안넣으면 됨.
	BannerName  string `json:"BannerName"`
	ImageURLPC  string `json:"ImageUrl_PC"`
	ImageURLM   string `json:"ImageUrl_M"`
	Description string `json:"description"`
	LinkURL     string `json:"linkURL"`
	Option      string `json:"option"`
	SortNum     int    `json:"sortNum"`
	Status      int    `json:"status"`
}
type GotchuPageShopOrderList struct {
	PageID   string `json:"pageId"`   //pageShopOrderList pageId
	Contents string `json:"contents"` //검색 옵션으로 받은 값
	Selected int    `json:"selected"` //위 검색 옵션이 어떤 타입인지 구별하는 값
	First    int    `json:"first"`
	Skip     int    `json:"skip"`
	OrderBy  string `json:"orderBy"`
	MaxDate  string `json:"maxDate"`
	MinDate  string `json:"minDate"`
}
type GotchuUsers struct {
	First             int    `json:"first"`
	Skip              int    `json:"skip"`
	MaxDate           string `json:"maxDate"`
	MinDate           string `json:"minDate"`
	ContainedNickname string `json:"containedNickname"`
	ContainedEmail    string `json:"containedEmail"`
	SubEmailKey       int    `json:"subEmailKey"` //0 true||false , 1 true , 2 false
}
type GotchuUserDetail struct {
	ID string `json:"id"`
}
type GotchuWebCashUpsert struct {
	ID    string `json:"id"`    //user Id
	Value int    `json:"value"` //증감할 금액 양.
	Memo  string `json:"memo"`  //memo (현재 필수 값이더라)
}
type GotchuInfluPageManage struct {
	MainPageID string `json:"mainPageId"`
	SubPageID  string `json:"subPageId"`
	Meta       string `json:"meta"`       //해당 구조체를 사용하는 api에서 처리뒤 기록을 남길경우 이 값을 주면 적제
	Option     string `json:"option"`     //해당 구조체를 사용하는 api에서 추가적인 필드를 넣고싶진 않지만 디테일한 처리 옵션을 넣고 싶다면 여기 기입.
	BeforeData string `json:"beforeData"` //main sub 통합전 데이터.
}
type GotchuFirebaseAuth struct {
	IDToken    string `json:"idToken"`
	ProviderID string `json:"providerId"`
	OTPValue   string `json:"otpValue"`
}
type GotchuFirebaseAuthClaim struct {
	FirebaseUID string   `json:"firebaseUID"`
	ID          string   `json:"id"`     //user Id
	Email       string   `json:"email"`  //user email
	Claims      []string `json:"claims"` //여기다 값넣어서 여러 클래임 먹이기도 가능. 지금은 안씀 어드민 부여만 하니까.
}
type TimeInterval struct {
	MaxDate string `json:"maxDate"`
	MinDate string `json:"minDate"`
}
type TimeTable struct {
	Columns    []string            `json:"columns"`
	Data       []map[string]string `json:"data"`
	Data2      []map[string]string `json:"data2"`
	DataCount  int                 `json:"dataCount"`
	Data2Count int                 `json:"data2Count"`
}
type InfluPageStruct struct {
	Size    int64  `json:"size"`
	Skip    int    `json:"skip"`
	MaxDate string `json:"maxDate"`
	MinDate string `json:"minDate"`
}
type InfluPageReturnDatas struct {
	TotalItemCount int64                 `json:"totalItemCount"`
	Maximums       string                `json:"maximums"`
	Minimums       string                `json:"minimums"`
	Rows           []InfluPageReturnData `json:"rows"`
}
type InfluPageReturnData struct {
	Number    int    `json:"size"`      //현재 순위로 사용.
	Contents  string `json:"contents"`  //인플루언서 네임 / 뱃지
	ViewCount int    `json:"viewCount"` //조회수
	PageID    string `json:"pageId"`
}
type CustomPage struct {
	ID             string                 `json:"id"`
	PageID         string                 `json:"pageId"`
	NickName       string                 `json:"nickName"`
	PageBadge      []PageBadge            `json:"badges"`
	AvatarURL      string                 `json:"avatarUrl"`
	CoverURL       string                 `json:"coverUrl"`
	Description    string                 `json:"description"`
	Owner          CustomUser             `json:"owner"`
	Youtube        *CustomYoutube         `json:"youtube"`
	Twitch         *CustomTwitch          `json:"twitch"`
	Instagram      *CustomInstagram       `json:"instagram"`
	AfreecaTV      *CustomAfreecaTV       `json:"afreecaTV"`
	Comments       []CustomComments       `json:"comments"`
	PageFanBoards  []PageFanBoard         `json:"pageFanBoard"`
	RelatedReviews []CustomRelatedReviews `json:"relatedReviews"`
	Fans           []CustomFans           `json:"fans"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
}
type CustomPage2 struct {
	ID          string `json:"id"`
	PageID      string `json:"pageId"`
	NickName    string `json:"nickName"`
	AvatarURL   string `json:"avatarUrl"`
	CoverURL    string `json:"coverUrl"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type PageBadge struct {
	ID    string `json:"id"`
	Vote  int    `json:"vote"`
	Badge Badge  `json:"badge"`
}
type Badge struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ImageURL    *string `json:"imageUrl,omitempty"`
	Description string  `json:"description"`
	OrderIndex  int32   `json:"orderIndex"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}
type CustomUser struct {
	ID                  string              `json:"id"`
	FirebaseUID         string              `json:"firebaseUID"`
	NickName            string              `json:"nickName"`
	Email               string              `json:"email"`
	OtpKey              string              `json:"otpKey"`
	DeletedAt           string              `json:"deletedAt"`
	CreatedAt           string              `json:"createdAt"`
	UpdatedAt           string              `json:"updatedAt"`
	Status              int                 `json:"status"`
	SubscribeEmail      bool                `json:"subscribeEmail"`
	CustomCashHistories []CustomCashHistory `json:"cashHistories"`
	CustomCoin          CustomCoin          `json:"coin"`
	CustomPage          CustomPage2         `json:"page"`
}
type CustomCoin struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Status          int     `json:"status"`
	Qty             float32 `json:"qty"`
	ContractAddress string  `json:"contractAddress"`
	DeployTxhash    string  `json:"deployTxhash"`
}
type CustomOrderHistory struct {
	Id          string     `json:"id"`
	Description string     `json:"description"`
	Memo        string     `json:"memo"`
	OrderPrice  float32    `json:"orderPrice"`
	DealPrice   float32    `json:"dealPrice"`
	OrderQty    float32    `json:"orderQty"`
	DealQty     float32    `json:"dealQty"`
	LeftQty     float32    `json:"leftQty"`
	TakerFee    float32    `json:"takerFee"`
	MakerFee    float32    `json:"makerFee"`
	DealFee     float32    `json:"dealFee"`
	OrderNum    int        `json:"orderNum"`
	Type        int        `json:"type"`
	IsCancel    bool       `json:"isCancel"`
	User        CustomUser `json:"user"`
	Coin        CustomCoin `json:"coin"`
}
type CustomCashHistory struct {
	Type     int32    `json:"type"`
	Property *int32   `json:"property,omitempty"`
	OPrice   *float64 `json:"oPrice,omitempty"`
	Qty      *float64 `json:"qty,omitempty"`
}
type CustomYoutube struct {
	ID string `json:"id"`
}
type CustomTwitch struct {
	ID string `json:"id"`
}
type CustomInstagram struct {
	ID string `json:"id"`
}
type CustomAfreecaTV struct {
	ID string `json:"id"`
}
type CustomComments struct {
	ID string `json:"id"`
}
type PageFanBoard struct {
	ID string `json:"id"`
}
type CustomRelatedReviews struct {
	ID string `json:"id"`
}
type CustomFans struct {
	ID string `json:"id"`
}
type UserListReturnData struct {
	ID             string  `json:"id"`
	NickName       string  `json:"nickName"`
	PageNickName   string  `json:"pageNickName"`
	Email          string  `json:"email"`
	CreatedAt      string  `json:"createdAt"`
	SubscribeEmail bool    `json:"subscribeEmail"`
	DepositSum     float64 `json:"depositSum"`
	WithdrawSum    float64 `json:"withdrawSum"`
}
type DeactivateUserListReturnData struct {
	ID             string  `json:"id"`
	NickName       string  `json:"nickName"`
	Email          string  `json:"email"`
	DeletedAt      string  `json:"deletedAt"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
	Status         int     `json:"status"`
	SubscribeEmail bool    `json:"subscribeEmail"`
	DepositSum     float64 `json:"depositSum"`
	EDD            string  `json:"EDD"`
}
type PrismaConnection struct {
	Aggregate Aggregate `json:"aggregate"`
}
type Aggregate struct {
	Count int `json:"count"`
}
type GWCResult struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type YouTubePageInfo struct {
	Items []YouTubePageInfoItem `json:"items"`
}

type YouTubePageInfoItem struct {
	ID               string                               `json:"id"`
	Kind             string                               `json:"kind"`
	Snippet          YoutTubePageInfoItemSnippet          `json:"snippet"`
	Statistics       YoutTubePageInfoItemStatistics       `json:"statistics"`
	BrandingSettings YoutTubePageInfoItemBrandingSettings `json:"brandingSettings"`
}

type YoutTubePageInfoItemSnippet struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	PublishedAt string            `json:"publishedAt"`
	Country     string            `json:"country"`
	Thumbnails  YouTubeThumbnails `json:"thumbnails"`
}
type YouTubeThumbnails struct {
	High YouTubeTHumbnailHigh `json:"high"`
}
type YouTubeTHumbnailHigh struct {
	URL string `json:"url"`
}
type YoutTubePageInfoItemStatistics struct {
	VideoCount      string `json:"videoCount"`
	SubscriberCount string `json:"subscriberCount"`
	ViewCount       string `json:"viewCount"`
}
type YoutTubePageInfoItemBrandingSettings struct {
	Image YouTubeBanner `json:"image"`
}
type YouTubeBanner struct {
	BannerImageURL string `json:"bannerImageUrl"`
}

type InstagramInfo struct {
	EntryData InstagramEntryData `json:"entry_data"`
}
type InstagramEntryData struct {
	ProfilePage []InstagramProfilePage `json:"ProfilePage"`
}
type InstagramProfilePage struct {
	Graphql InstagramGraphql `json:"graphql"`
}
type InstagramGraphql struct {
	User InstagramUser `json:"user"`
}
type InstagramUser struct {
	ID            string             `json:"id"`
	UserName      string             `json:"username"`
	FullName      string             `json:"full_name"`
	Post          InstagramPost      `json:"edge_owner_to_timeline_media"`
	Follower      InstagramFollower  `json:"edge_followed_by"`
	Following     InstagramFollowing `json:"edge_follow"`
	ProfilePicURL string             `json:"profile_pic_url"`
	Biography     string             `json:"biography"`
}
type InstagramPost struct {
	Count int `json:"count"`
}
type InstagramFollower struct {
	Count int `json:"count"`
}
type InstagramFollowing struct {
	Count int `json:"count"`
}

type TwitchUserInfoResult struct {
	Data []TwitchUserInfo `json:"data"`
}
type TwitchUserInfo struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
}

type AfreecaResult struct {
	ProfileImage string         `json:"profile_image"`
	Station      AfreecaStation `json:"station"`
}
type AfreecaStation struct {
	UserID       string         `json:"user_id"`
	UserNick     string         `json:"user_nick"`
	StationNo    int            `json:"station_no"`
	StationName  string         `json:"station_name"`
	StationTitle string         `json:"station_title"`
	Upd          AfreecaUpd     `json:"upd"`
	Display      AfreecaDisplay `json:"display"`
}
type AfreecaUpd struct {
	FanCnt        int `json:"fan_cnt"`
	TotalViewCnt  int `json:"total_view_cnt"`
	TotalVisitCnt int `json:"total_visit_cnt"`
}
type AfreecaDisplay struct {
	ProfileText string `json:"profile_text"`
}

func main() {
	if runtime.GOOS == "darwin" {
		gaKeyPath = "gaKey.json"
		firebaseKeyPath = "firebase-adminsdk.json" //live
		// firebaseKeyPath = "" //dev
	} else {
		gaKeyPath = "/home/ec2-user/go/src/gotchu.admin.server/gaKey.json"
		firebaseKeyPath = "/home/ec2-user/go/src/gotchu.admin.server/firebase-adminsdk.json" //live
		// firebaseKeyPath = "" //dev
	}
	initFirebaseSDK() //firebase sdk
	connectAWS()      //aws sdk
	setGA()           //google analytics

	pClient = prisma.New(nil)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	//cors set
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://admin.gotchu.io", "http://admin.gotchu.io"} //for live
	config.AllowHeaders = []string{"Origin", "m", "Accept", "Content-Type", "set-cookie"}
	config.AllowCredentials = true
	r.Use(cors.New(config))
	//cors set end
	r.Use(sessions.Sessions("aa", sessions.NewCookieStore([]byte("secret")))) //admin auth
	r.POST("/login", login)
	r.POST("/rno", rno) //request new otp
	r.POST("/otpCheck", otpCheck)
	r.GET("/test1", test1)
	r.GET("/test2", test2)
	r.GET("/logout", logout)
	r.GET("/ping", ping)

	private := r.Group("/private")
	private.Use(AuthRequired)
	{
		private.POST("/addAdminClaim", addAdminClaim) //어드민 권한을 부여하는 api
		private.POST("/otpInit", otpInit)             //특정 유저의 otp 값을 초기화 하는 api
		private.POST("/gfi", gfi)                     //get firebase info 특정 유저의 파이어베이스 정보를 확인가능(어드민 권한, 오티피 난수 등)
		private.GET("/gfaul", gfaul)                  //get firebase admin user list 임시로 간략하게 보여줄 예정
		private.GET("/me", me)                        //session check
		private.GET("/status", status)                //session status
		// private.POST("/postlist", postlist)
		private.POST("/gaVPData", gaVPData)                         //방문자 수 / 페이지 뷰 조회 관련 api
		private.POST("/influPageView", influPageView)               //인플루언서페이지 조회 관련 api
		private.GET("/cdaw", cdaw)                                  //CumulativeDepositAndWithdrawal
		private.POST("/gotchuCashList", gotchuCashList)             //갓츄 캐시리스트
		private.GET("/memberCount", memberCount)                    //갓츄 회원수 (앱가입, 웹가입, 전체)
		private.POST("/msw", msw)                                   //memberSignWithdraw 특정 달의 가입 회원/탈퇴 회원 정보 제공.
		private.POST("/userList", userList)                         //user List
		private.POST("/userDetail", userDetail)                     //user Detail
		private.POST("/deactivateUserList", deactivateUserList)     //deactivateUser Detail
		private.POST("/makeDeactivatingUser", makeDeactivatingUser) //탈퇴 처리 예정인 유저로 만듬.
		private.POST("/makeDeactivatedUser", makeDeactivatedUser)   //탈퇴 처리 예정 유저의 개인정보 삭제후 탈퇴 완료로 만듬.
		private.GET("/searchInfluPage", searchInfluPage)            //인플루언서 페이지 찾기 api
		private.POST("/compareInfluPages", compareInfluPages)       //통합하려는 인플루언서 페이지 둘 데이터 제공
		private.POST("/mergeInfluPages", mergeInfluPages)           //두 인플루언서 페이지를 통합함.
		private.POST("/snsAcountLinkList", snsAcountLinkList)       //snsAcountLinkList
		private.POST("/salp", salp)                                 //snsAcountLinkProcessing
		private.POST("/refuseSNSLink", refuseSNSLink)               //snsAcounrLink refuse
		private.POST("/reviewList", reviewList)                     //review List
		private.POST("/deleteReview", deleteReview)                 //delete review
		private.GET("/ticketInfo1", ticketInfo1)                    //ticketInfo1 총 티켓판매금액, 티켓 판매해본 크리에이터 수, 티켓 구매건 수, 구매해본 유저 수 넘김.
		private.POST("/ticketUserList", ticketUserList)             //티켓 유저 관리 화면에서 보여주는 유저 리스트
		private.POST("/ticketUserDetail", ticketUserDetail)         //티켓 유저 상세에서 유저정보와 현재티켓 정보 반환하는 api
		private.POST("/pageShopOrderList", pageShopOrderList)       //티켓 유저 상세와 구매내역에서 유저정보와 현재티켓 정보 반환하는 api
		private.POST("/updateUseShopM", updateUseShopM)             //티켓 유저 상세에서 유저 키를 넘겨주면 상태를 뒤집어줌 api
		private.POST("/uploadImage", uploadImage)                   //파일을 업로드 하고, 이미지주소를 반환하는 api. 배너때문에 추가됨.
		private.POST("/upsertBanner", upsertBanner)                 //배너를 새로 만들거나, 업데이트함. 배너네임 기준으로 처리.
		private.GET("/webBannerList", webBannerList)                //웹배너 리스트를 보여줌. 활성화,비활성화만 보여주고 논리삭제는 안보여줌.
		private.POST("/upsertPSOC", upsertPSOC)                     //PageShopOrder의 PageShopOrderConfirmation을 생성하거나 업데이트함.
		private.POST("/adminWebCashChange", adminWebCashChange)     //어드민에서 웹캐시 증감(캐시히스토리 생성)
	}
	ctx = context.Background()
	//
	r.Run(":8080")
	// r.Run(":8088") //for dev
}

func p(err error) {
	if err != nil {
		panic(err)
	}
}

func setGA() {
	key, _ := ioutil.ReadFile(gaKeyPath)

	jwtConf, err := google.JWTConfigFromJSON(
		key,
		analytics.AnalyticsReadonlyScope,
	)
	p(err)

	httpClient := jwtConf.Client(oauth2.NoContext)
	garService, err = ga.New(httpClient)
	p(err)
}

func initFirebaseSDK() {
	opt := option.WithCredentialsFile(firebaseKeyPath)
	var err error
	firebaseApp, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		sendSlack(err, "initFirebaseSDK err >> ")
	}
}

func connectAWS() {
	var err error
	awsSession, err = session.NewSession(
		&aws.Config{
			Region:      aws.String("ap-northeast-2"),
			Credentials: credentials.NewStaticCredentials(AWS_ACCESS_KEY_ID, AWS_ACCESS_KEY, ""),
		},
	)
	if err != nil {
		panic(err)
	}
}

func deleteFirebaseUser(uid string) bool {
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "deleteFirebaseUser Auth err >> ")
		return false
	}
	err2 := client.DeleteUser(context.Background(), uid)
	if err2 != nil {
		sendSlack(err2, "deleteFirebaseUser deleting user err >> ")
		if err2.Error() == "googleapi: Error 400: USER_NOT_FOUND, invalid" {
			return true
		}
		return false
	}
	// sendSlack(uid, "Successfully deleted user >> ")
	return true
}

func gwcConnector(apiPath string, urlEncodedParamStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", gwcServerURL+apiPath+"?"+urlEncodedParamStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func stibeeConnector(body string, method string) (*http.Response, error) {
	req, err := http.NewRequest(method, stibeeSubscribeURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AccessToken", stibeeAccessToken)

	client := &http.Client{}
	return client.Do(req)
}

func youTubeChannelInfoConnector(urlEncodedParamStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", youtubeChannelURL+"?"+urlEncodedParamStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func instagramConnector(userName string) (InstagramUser, error) { //get instagram user info by userName
	var instaUser InstagramUser
	ig, err := gista.New(nil)
	if err != nil {
		return instaUser, err
	}
	username, password := "hey@gotchu.io", "plbkn37X"
	err = ig.Login(username, password, false)
	if err != nil {
		return instaUser, err
	}
	var module *string
	result, err := ig.People.GetInfoByName(userName, module)
	if err != nil {
		return instaUser, err
	}
	// fmt.Println(result.User.Username)
	// fmt.Println(result.User.UserId)
	// fmt.Println(result.User.FullName)
	// fmt.Println(result.User.ProfilePicId)
	// fmt.Println(result.User.Id)
	// fmt.Println(result.User.PageId)
	// fmt.Println(result.User.ProfilePicUrl)
	// fmt.Println(result.User.MediaCount)
	// fmt.Println(result.User.Biography)
	// fmt.Println(result.User.FollowerCount)
	// fmt.Println(result.User.FollowingCount)
	instaUser.Biography = result.User.Biography
	instaUser.Follower.Count = result.User.FollowerCount
	instaUser.Following.Count = result.User.FollowingCount
	instaUser.FullName = result.User.FullName
	instaUser.ID = strings.Split(result.User.ProfilePicId, "_")[1]
	instaUser.Post.Count = result.User.MediaCount
	instaUser.ProfilePicURL = result.User.ProfilePicUrl
	instaUser.UserName = result.User.Username
	return instaUser, err

	// c := colly.NewCollector()
	// var fuck string
	// var rerr error
	// var visitURL string
	// visitURL = "https://www.instagram.com/" + userName + "/?__a=1"
	// c.OnResponse(func(r *colly.Response) {
	// 	// fmt.Println("resp >> ", string(r.Body))
	// 	fuck = string(r.Body)
	// })
	// c.Request("GET",
	// 	visitURL,
	// 	strings.NewReader(`term=4404094910003`),
	// 	nil,
	// 	http.Header{"Cookie": []string{cookieStr}},
	// )
	// // c.OnHTML("script", func(e *colly.HTMLElement) {
	// // 	fmt.Println("e.Text >> ", e.Text)
	// // 	if strings.Contains(e.Text, `{"config":{"csrf_token"`) {
	// // 		fuck = e.Text[21 : len(e.Text)-1]
	// // 	}
	// // })
	// c.OnError(func(r *colly.Response, err error) {
	// 	fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	// 	fuck = string(r.Body)
	// 	rerr = err
	// })
	// // c.Visit(visitURL)
	// c.Wait()

	// var data InstagramProfilePage
	// json.Unmarshal([]byte(fuck), &data)
	// // fmt.Println("data >> ", data)

	// return data, rerr
}

func twitchUserInfoConnector(userID string) (*http.Response, error) {
	req, err := http.NewRequest("GET", twitchUserInfoURL+"?login="+userID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", "")
	req.Header.Set("Authorization", "Bearer ")

	client := &http.Client{}
	return client.Do(req)
}

func twitchUserFollowConnector(userID string) (followerCount float64, followingCount float64) {
	req1, err := http.NewRequest("GET", twitchUserFollowURL+"?first=1&to_id="+userID, nil)
	if err != nil {
		followerCount = 0
	}
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Client-Id", "")
	req1.Header.Set("Authorization", "Bearer ")
	client1 := &http.Client{}
	res1, err := client1.Do(req1)
	if err != nil {
		followerCount = 0
	}
	if res1.StatusCode != 200 {
		followerCount = 0
	}
	lBody1, _ := ioutil.ReadAll(res1.Body)
	var result1 map[string]interface{}
	json.Unmarshal(lBody1, &result1)
	followerCount = result1["total"].(float64)

	req2, err := http.NewRequest("GET", twitchUserFollowURL+"?first=1&from_id="+userID, nil)
	if err != nil {
		followingCount = 0
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Client-Id", "")
	req2.Header.Set("Authorization", "Bearer ")
	client2 := &http.Client{}
	res2, err := client2.Do(req2)
	if err != nil {
		followingCount = 0
	}
	if res2.StatusCode != 200 {
		followingCount = 0
	}
	lBody2, _ := ioutil.ReadAll(res2.Body)
	var result2 map[string]interface{}
	json.Unmarshal(lBody2, &result2)
	followingCount = result2["total"].(float64)
	return followerCount, followingCount
}

func afreecaConnector(userID string) (*http.Response, error) {
	req, err := http.NewRequest("GET", afreecatvAPIURL+userID+"/station", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.132 Safari/537.36")

	client := &http.Client{}
	return client.Do(req)
}

func sendSlack(message interface{}, flagStr string) {
	hookType := 0 //어떤 슬랙채널에 전달할지 판단.
	var str strings.Builder
	str.WriteString(`{"text": "`)
	str.WriteString(PROJECTNAME)
	str.WriteString(" // ")
	str.WriteString(flagStr)
	switch v := message.(type) {
	case error:
		str.WriteString(v.Error())
		hookType = 1
	case string:
		str.WriteString(strings.Replace(strings.Replace(strings.Replace(v, "\\", "", -1), "\n", "", -1), "\"", "\\\"", -1))
	default:
		jsonStr, err := json.Marshal(message)
		if err != nil {
			str.WriteString(fmt.Sprint(message))
			str.WriteString(`__json.Marshal err__`)
			str.WriteString(fmt.Sprint(err))
		} else {
			jstr := strconv.Quote(string(jsonStr))
			jstr = jstr[1 : len(jstr)-1]
			str.WriteString(jstr)
		}
	}
	str.WriteString(`"}`)
	var strS = str.String()
	fmt.Println("sendSlack strS >> ", strS)
	slackHookURI := slackHook
	if hookType == 0 {
		slackHookURI = slackHookNoti
	}
	lReq, err := http.NewRequest("POST", slackHookURI, strings.NewReader(strS))
	lReq.Header.Set("Content-Type", "application/json")
	lClient := &http.Client{}
	lResp, err := lClient.Do(lReq)
	if err != nil {
		log.Println("sendSlack >> ", err)
		return
	}
	defer lResp.Body.Close()
}

func removeSC(s string) string { //remove special characters
	return strings.Replace(strings.Replace(strings.Replace(strings.Replace(s, "\\", "", -1), "\n", "", -1), "\r", "", -1), "\"", "\\\"", -1)
}

// AuthRequired is a simple middleware to check the session
func AuthRequired(c *gin.Context) {
	cTime := time.Now().UnixNano()
	session := sessions.Default(c)
	m := c.GetHeader("m")
	user := session.Get(m)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userTime := user.(int64)
	if cTime-userTime > sessionLifeTime {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

// login is a handler that parses a form and checks for specific data
func login(c *gin.Context) {
	var json GotchuFirebaseAuth
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	// firebase auth check
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "login firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login firebaseApp err"})
		return
	}
	tokenResult, err := client.VerifyIDToken(context.Background(), json.IDToken)
	if err != nil {
		sendSlack(err, "login VerifyIDToken err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login VerifyIDToken err"})
		return
	}
	// firebase auth check success
	// prisma user check
	user, err := pClient.User(prisma.UserWhereUniqueInput{
		FirebaseUid: &tokenResult.UID,
	}).Exec(ctx)
	if err != nil {
		sendSlack(err, "login prisma user query err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	if user == nil {
		sendSlack(err, "login prisma user not found >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	if user.Provider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider null"})
		return
	} else if *user.Provider != json.ProviderID {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provider does not match"})
		return
	}
	// fmt.Println(user)
	//prisma user check end
	// custom claim check
	claims := tokenResult.Claims
	if admin, ok := claims["admin"]; ok {
		if admin.(bool) {
			//check has otp >> true - frontend occur otpvalue , false - request new otp
			userEmail := fmt.Sprint(claims["email"])
			if otpKey, ok := claims["otpKey"]; ok {
				otpScretKey := fmt.Sprint(otpKey)
				if otpScretKey != "" {
					c.JSON(http.StatusOK, gin.H{"success": true, "hasOTP": true, "accessEmail": userEmail})
					return
				} else {
					c.JSON(http.StatusOK, gin.H{"success": true, "hasOTP": false, "accessEmail": userEmail})
					return
				}
			} else {
				c.JSON(http.StatusOK, gin.H{"success": true, "hasOTP": false, "accessEmail": userEmail})
				return
			}
		}
	} else {
		sendSlack(claims["email"], "Who are you? >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Who are you?"})
		return
	}
	// fmt.Println(tokenResult.Audience)
	// fmt.Println(tokenResult.UID) // firebase uid (prisma db 에서 유저 쿼리해오는데 씀)
	// fmt.Println(tokenResult.Subject)
	// fmt.Println(tokenResult.Issuer)
	// fmt.Println(tokenResult.IssuedAt)
	// fmt.Println(tokenResult.Expires)
	// fmt.Println(tokenResult.Claims)
}

func addAdminClaim(c *gin.Context) {
	var json GotchuFirebaseAuthClaim
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var firebaseUID string
	if json.FirebaseUID == "" {
		if json.ID == "" {
			if json.Email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "FUID and UserID and email empty"})
				return
			} else {
				existCheckQuery := `{
					user(where:{email:"` + json.Email + `"}){
						firebaseUID
					}
				  }`
				ecVariables := make(map[string]interface{})
				ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				firebaseUID = ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
			}
		} else {
			existCheckQuery := `{
				user(where:{id:"` + json.ID + `"}){
					firebaseUID
				}
			  }`
			ecVariables := make(map[string]interface{})
			ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			firebaseUID = ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
		}
	} else {
		firebaseUID = json.FirebaseUID
	}
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "addAdminClaim firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "addAdminClaim firebaseApp err"})
		return
	}
	user, err := client.GetUser(ctx, firebaseUID)
	if err != nil {
		log.Fatal(err)
	}
	newCustomClaims := user.CustomClaims
	if len(newCustomClaims) == 0 {
		newCustomClaims = make(map[string]interface{}, 1)
	}
	newCustomClaims["admin"] = true
	err = client.SetCustomUserClaims(ctx, firebaseUID, newCustomClaims)
	if err != nil {
		sendSlack(err, "addAdminClaim setting custom claims error >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "addAdminClaim setting custom claims error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func otpInit(c *gin.Context) {
	var json GotchuFirebaseAuthClaim
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var firebaseUID string
	if json.FirebaseUID == "" {
		if json.ID == "" {
			if json.Email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "FUID and UserID and email empty"})
				return
			} else {
				existCheckQuery := `{
					user(where:{email:"` + json.Email + `"}){
						firebaseUID
					}
				  }`
				ecVariables := make(map[string]interface{})
				ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				firebaseUID = ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
			}
		} else {
			existCheckQuery := `{
				user(where:{id:"` + json.ID + `"}){
					firebaseUID
				}
			  }`
			ecVariables := make(map[string]interface{})
			ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			firebaseUID = ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
		}
	} else {
		firebaseUID = json.FirebaseUID
	}
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "otpInit firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "otpInit firebaseApp err"})
		return
	}
	user, err := client.GetUser(ctx, firebaseUID)
	if err != nil {
		sendSlack(err, "otpInit firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetUser err"})
		return
	}
	newCustomClaims := make(map[string]interface{})
	if len(user.CustomClaims) != 0 {
		newCustomClaims = user.CustomClaims
	}
	newCustomClaims["otpKey"] = ""
	err = client.SetCustomUserClaims(ctx, firebaseUID, newCustomClaims)
	if err != nil {
		sendSlack(err, "otpInit setting custom claims error >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "otpInit setting custom claims error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func gfi(c *gin.Context) {
	var json GotchuFirebaseAuthClaim
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var firebaseUID string
	if json.FirebaseUID == "" {
		if json.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "FUID and UserID empty"})
			return
		} else {
			existCheckQuery := `{
				user(where:{id:"` + json.ID + `"}){
					firebaseUID
				}
			  }`
			ecVariables := make(map[string]interface{})
			ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			firebaseUID = ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
		}
	} else {
		firebaseUID = json.FirebaseUID
	}
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "gfi firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gfi firebaseApp err"})
		return
	}
	user, err := client.GetUser(ctx, firebaseUID)
	if err != nil {
		sendSlack(err, "gfi firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetUser err"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": user})
}

func gfaul(c *gin.Context) {
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "gfaul firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gfaul firebaseApp err"})
		return
	}
	var userList []*auth.ExportedUserRecord

	pager := iterator.NewPager(client.Users(ctx, ""), 1000, "")
	for {
		var users []*auth.ExportedUserRecord
		nextPageToken, err := pager.NextPage(&users)
		if err != nil {
			sendSlack(err, "gfaul paging error >> ")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gfaul paging error"})
			return
		}
		for _, u := range users {
			if v, b := u.CustomClaims["admin"]; b {
				if v.(bool) {
					userList = append(userList, u)
				}
			}
		}
		if nextPageToken == "" {
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{"result": userList})
}

func rno(c *gin.Context) {
	var json GotchuFirebaseAuth
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// firebase auth check
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "rno firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rno firebaseApp err"})
		return
	}
	tokenResult, err := client.VerifyIDToken(context.Background(), json.IDToken)
	if err != nil {
		sendSlack(err, "rno VerifyIDToken err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rno VerifyIDToken err"})
		return
	}
	// prisma user check
	user, err := pClient.User(prisma.UserWhereUniqueInput{
		FirebaseUid: &tokenResult.UID,
	}).Exec(ctx)
	if err != nil {
		sendSlack(err, "rno prisma user query err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	if user == nil {
		sendSlack(err, "rno prisma user not found >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	//prisma user check end
	//check customclaims otpkey
	checkClaims := tokenResult.Claims
	if otpKey, ok := checkClaims["otpKey"]; ok {
		otpScretKey := fmt.Sprint(otpKey)
		if otpScretKey != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP already issued"})
			return
		}
	}
	fmt.Println(checkClaims)
	//create otp uri and save scretkey on db
	secretLength := 16
	secretStr := gotp.RandomSecret(secretLength)
	firebaseUser, err := client.GetUser(ctx, tokenResult.UID)
	if err != nil {
		log.Fatal(err)
	}
	newCustomClaims := firebaseUser.CustomClaims
	if len(newCustomClaims) == 0 {
		newCustomClaims = make(map[string]interface{}, 1)
	}
	newCustomClaims["otpKey"] = secretStr
	err = client.SetCustomUserClaims(ctx, tokenResult.UID, newCustomClaims)
	if err != nil {
		sendSlack(err, "rno setting custom claims error >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rno setting custom claims error"})
		return
	}
	userEmail := fmt.Sprint(checkClaims["email"])
	otpURI := gotp.NewDefaultTOTP(secretStr).ProvisioningUri(userEmail, "Gotchu Admin")
	sendSlack(userEmail, "어드민에 새로 OTP발급을 받음 >> ")
	c.JSON(http.StatusOK, gin.H{"otpURI": otpURI})
}

func otpCheck(c *gin.Context) {
	session := sessions.Default(c)

	var json GotchuFirebaseAuth
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// firebase auth check
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "otpCheck firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "otpCheck firebaseApp err"})
		return
	}
	tokenResult, err := client.VerifyIDToken(context.Background(), json.IDToken)
	if err != nil {
		sendSlack(err, "otpCheck VerifyIDToken err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "otpCheck VerifyIDToken err"})
		return
	}
	// prisma user check
	user, err := pClient.User(prisma.UserWhereUniqueInput{
		FirebaseUid: &tokenResult.UID,
	}).Exec(ctx)
	if err != nil {
		sendSlack(err, "otpCheck prisma user query err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	if user == nil {
		sendSlack(err, "otpCheck prisma user not found >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FirebaseUid not found in db"})
		return
	}
	//prisma user check end
	//check otpkey
	checkClaims := tokenResult.Claims
	userEmail := fmt.Sprint(checkClaims["email"])
	if otpKey, ok := checkClaims["otpKey"]; ok {
		otpScretKey := fmt.Sprint(otpKey)
		if otpScretKey != "" {
			notp := gotp.NewDefaultTOTP(otpScretKey).Now()
			if notp == json.OTPValue {
				session.Set(userEmail, time.Now().UnixNano())
				if err := session.Save(); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
					return
				}
				// fmt.Println(session.Get(json.AccessEmail))
				sendSlack(userEmail, "어드민에 로그인 함 >> ")
				c.JSON(http.StatusOK, gin.H{"success": true, "accessEmail": userEmail})
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": false, "accessEmail": userEmail})
	return
}

func uploadFile(filename string, file io.Reader) (imageURL string) {
	uploader := s3manager.NewUploader(awsSession)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(AWS_BUCKET_NAME),
		Key:    aws.String(AWS_BUCKET_FOLDER_NAME + filename),
		Body:   file,
	})
	if err != nil {
		// fmt.Printf("Unable to upload %q to %q, %v", filename, AWS_BUCKET_FOLDER_NAME, err)
		sendSlack(err, "Unable to upload "+filename+" // "+AWS_BUCKET_FOLDER_NAME)
	}
	return result.Location
}

func test1(c *gin.Context) {
	var json GotchuFirebaseAuthClaim
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "addAdminClaim firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "addAdminClaim firebaseApp err"})
		return
	}
	user, err := client.GetUser(ctx, json.FirebaseUID)
	if err != nil {
		sendSlack(err, "addAdminClaim firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetUser err"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": user})
}

func test2(c *gin.Context) {
	var json GotchuFirebaseAuthClaim
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, err := firebaseApp.Auth(context.Background())
	if err != nil {
		sendSlack(err, "addAdminClaim firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "addAdminClaim firebaseApp err"})
		return
	}
	user, err := client.GetUser(ctx, json.FirebaseUID)
	if err != nil {
		sendSlack(err, "addAdminClaim firebaseApp err >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetUser err"})
		return
	}
	newCustomClaims := make(map[string]interface{})
	if len(user.CustomClaims) != 0 {
		newCustomClaims = user.CustomClaims
	}
	newCustomClaims["otpKey"] = ""
	err = client.SetCustomUserClaims(ctx, json.FirebaseUID, newCustomClaims)
	if err != nil {
		sendSlack(err, "addAdminClaim setting custom claims error >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "addAdminClaim setting custom claims error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	m := c.GetHeader("m")
	user := session.Get(m)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(m)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func me(c *gin.Context) {
	session := sessions.Default(c)
	m := c.GetHeader("m")
	user := session.Get(m)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}

func gaVPData(c *gin.Context) {
	var jsonR TimeInterval //max만 사용함
	if err := c.ShouldBindJSON(&jsonR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := now.Parse(jsonR.MaxDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	btimeTime := now.With(t).BeginningOfMonth()
	bTime := btimeTime.Format(time.RFC3339)[0:10]
	// bTimeDay := bTime[8:10]
	eTime := now.With(t).EndOfMonth().Format(time.RFC3339)[0:10]
	eTimeDay, err := strconv.Atoi(eTime[8:10])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req := &ga.GetReportsRequest{
		// Our request contains only one request
		// So initialise the slice with one ga.ReportRequest object
		ReportRequests: []*ga.ReportRequest{
			// Create the ReportRequest object.
			{
				ViewId: viewID,
				DateRanges: []*ga.DateRange{
					// Create the DateRange object.
					{StartDate: bTime, EndDate: eTime},
				},
				Metrics: []*ga.Metric{
					// Create the Metrics object.
					{Expression: "ga:users"},
					{Expression: "ga:pageviews"},
				},
				Dimensions: []*ga.Dimension{
					{Name: "ga:date"},
					{Name: "ga:hour"},
				},
				OrderBys: []*ga.OrderBy{
					{FieldName: "ga:date", SortOrder: "DESCENDING"},
				},
				PageSize: gaVPDataPageSize,
			},
		},
	}
	res, err := garService.Reports.BatchGet(req).Do()
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "garService fail"})
		return
	}

	var tt TimeTable
	tt.Columns = append(tt.Columns, "날짜/시간")
	for i := 0; i < 24; i++ {
		tt.Columns = append(tt.Columns, fmt.Sprintf("%02d", i))
	}
	for i := 0; i < eTimeDay; i++ {
		defaultTTMap := make(map[string]string)
		for i := 0; i < 24; i++ {
			defaultTTMap[fmt.Sprintf("%02d", i)] = ""
		}
		dateStr := btimeTime.AddDate(0, 0, i).Format("06-01-02")
		defaultTTMap["날짜/시간"] = dateStr
		tt.Data = append(tt.Data, defaultTTMap)
		defaultTTMap2 := make(map[string]string)
		for i := 0; i < 24; i++ {
			defaultTTMap2[fmt.Sprintf("%02d", i)] = ""
		}
		dateStr2 := btimeTime.AddDate(0, 0, i).Format("06-01-02")
		defaultTTMap2["날짜/시간"] = dateStr2
		tt.Data2 = append(tt.Data2, defaultTTMap2)
	}
	for _, v := range res.Reports[0].Data.Rows {
		matchStick := v.Dimensions[0][2:4] + "-" + v.Dimensions[0][4:6] + "-" + v.Dimensions[0][6:8]
		matchStick2 := v.Dimensions[1]
		for _, vv := range tt.Data {
			if vv["날짜/시간"] == matchStick {
				vv[matchStick2] = v.Metrics[0].Values[0]
			}
		}
		for _, vv := range tt.Data2 {
			if vv["날짜/시간"] == matchStick {
				vv[matchStick2] = v.Metrics[0].Values[1]
			}
		}
	}
	dataCount, _ := strconv.Atoi(res.Reports[0].Data.Totals[0].Values[0])
	tt.DataCount = dataCount
	data2Count, _ := strconv.Atoi(res.Reports[0].Data.Totals[0].Values[1])
	tt.Data2Count = data2Count

	c.JSON(http.StatusOK, tt)
}

func influPageView(c *gin.Context) {
	var jsonR InfluPageStruct //max만 사용함
	if err := c.ShouldBindJSON(&jsonR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonR.MinDate == "" {
		jsonR.MinDate = gaDefaultStartDate
	}
	if jsonR.MaxDate == "" {
		jsonR.MaxDate = gaDefaultEndDate
	}
	req := &ga.GetReportsRequest{
		// Our request contains only one request
		// So initialise the slice with one ga.ReportRequest object
		ReportRequests: []*ga.ReportRequest{
			// Create the ReportRequest object.
			{
				ViewId: viewID,
				DateRanges: []*ga.DateRange{
					// Create the DateRange object.
					{StartDate: jsonR.MinDate, EndDate: jsonR.MaxDate},
				},
				Metrics: []*ga.Metric{
					// Create the Metrics object.
					{Expression: "ga:pageviews"},
				},
				Dimensions: []*ga.Dimension{
					{Name: "ga:pagePath"},
				},
				OrderBys: []*ga.OrderBy{
					{FieldName: "ga:pageviews", SortOrder: "DESCENDING"},
				},
				FiltersExpression: "ga:pagepath=@pages",
				PageSize:          jsonR.Size,
				PageToken:         strconv.Itoa(jsonR.Skip),
			},
		},
	}
	res, err := garService.Reports.BatchGet(req).Do()
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "garService fail"})
		return
	}
	if len(res.Reports) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetReportsRequest Reports empty"})
		return
	}
	if res.Reports[0].Data == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GetReportsRequest Reports.Data nil"})
		return
	}
	//reports데이터 가공
	var fResData InfluPageReturnDatas
	fResData.Rows = make([]InfluPageReturnData, len(res.Reports[0].Data.Rows))
	if len(res.Reports[0].Data.Maximums) == 0 {
		fResData.Maximums = "0"
	} else {
		fResData.Maximums = res.Reports[0].Data.Maximums[0].Values[0]
	}
	if len(res.Reports[0].Data.Minimums) == 0 {
		fResData.Minimums = "0"
	} else {
		fResData.Minimums = res.Reports[0].Data.Minimums[0].Values[0]
	}
	fResData.TotalItemCount = res.Reports[0].Data.RowCount
	pagesDataResult, err := getPrismaPagesData(res.Reports[0].Data.Rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var contentStr strings.Builder
	for k, v := range res.Reports[0].Data.Rows {
		fResData.Rows[k].Number = k
		viewCount, err := strconv.Atoi(v.Metrics[0].Values[0])
		if err != nil {
			viewCount = 0
		}
		fResData.Rows[k].ViewCount = viewCount
		pageID := v.Dimensions[0][7:]
		fResData.Rows[k].PageID = pageID
		for _, v := range pagesDataResult {
			if pageID == v.PageID {
				for _, v := range v.PageBadge {
					contentStr.WriteString("#")
					contentStr.WriteString(v.Badge.Name)
				}
				contentStr.WriteString(" \n")
				contentStr.WriteString(v.NickName)
				fResData.Rows[k].Contents = contentStr.String()
				contentStr.Reset()
				break
			}
		}
	}
	//reports데이터 가공 end
	c.JSON(http.StatusOK, fResData)
}

func getPrismaPagesData(rows []*ga.ReportRow) ([]CustomPage, error) {
	query := `
	query{
		pages(where:{OR:[`
	for k, v := range rows {
		pageStr := v.Dimensions[0]
		if k != 0 {
			query += `,`
		}
		query += `{pageId:"` + pageStr[7:] + `"}`
	}
	query += `]}){
		  pageId
		  nickName
		  badges(orderBy:vote_DESC,first:3){
			badge{
			  name
			}
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, err
	}
	pages := result["pages"].([]interface{})
	var fResult []CustomPage
	for _, v := range pages {
		jsonString, _ := json.Marshal(v)
		s := CustomPage{}
		json.Unmarshal(jsonString, &s)
		fResult = append(fResult, s)
	}
	return fResult, err
}

func cdaw(c *gin.Context) {
	query := `
	query{
		cashHistories(where:{type_in:[1,2],property_in:[1,2,11]}){
		 type,property,oPrice,qty
	   }
	}`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	values := result["cashHistories"]
	v, ok := values.([]interface{})
	// fmt.Println(v)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handle error"})
		return
	}
	deSum, wiSum := calDepositAndWithdrawSum(v)
	c.JSON(http.StatusOK, gin.H{"depositSum": deSum, "withdrawSum": wiSum})
}

func calDepositAndWithdrawSum(v []interface{}) (deSum float64, wiSum float64) {
	for _, v := range v {
		jsonString, _ := json.Marshal(v)

		s := prisma.CashHistory{}
		json.Unmarshal(jsonString, &s)
		if s.Type == 1 {
			if *s.Property == 1 || *s.Property == 11 {
				deSum += *s.Qty
			}
		} else if s.Type == 2 {
			if *s.Property == 2 {
				wiSum += *s.OPrice
			}
		}
	}
	return deSum, wiSum
}

func calDepositAndWithdrawSumStruct(v []CustomCashHistory) (deSum float64, wiSum float64) {
	for _, v := range v {
		if v.Type == 1 {
			if *v.Property == 1 || *v.Property == 11 {
				deSum += *v.Qty
			}
		} else if v.Type == 2 {
			if *v.Property == 2 {
				wiSum += *v.OPrice
			}
		}
	}
	return deSum, wiSum
}

func gotchuCashList(c *gin.Context) {
	var jsonS GotchuCashList
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `
	 query{
		cashHistories(where:{type_in:[1,2],property_in:[1,2,11]`
	if jsonS.MinDate != "" {
		query += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		query += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	query += `},orderBy:createdAt_DESC,first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `){
		 user{
		   nickName
		 }
		 updatedAt
		 type
		 property
		 memo
		 qty
		 oPrice
	   }
	   cashHistoriesConnection(where:{type_in:[1,2],property_in:[1,2,11]`
	if jsonS.MinDate != "" {
		query += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		query += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	query += `}){
		 aggregate{
		   count
		 }
	   }
	 }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func memberCount(c *gin.Context) {
	query := `
	query{
		R1:usersConnection(where:{platform_not:null,AND:[{email_not_contains:"yopmail.com"},{email_not_contains:"blocko.xyz"}]}){
		  aggregate{
			count
		  }
		}
		R2:usersConnection(where:{platform:null,AND:[{email_not_contains:"yopmail.com"},{email_not_contains:"blocko.xyz"}]}){
		  aggregate{
			count
		  }
		}
		R3:usersConnection(where:{AND:[{email_not_contains:"yopmail.com"},{email_not_contains:"blocko.xyz"}]}){
		  aggregate{
			count
		  }
		}
	}`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func msw(c *gin.Context) { //memberSignWithdraw
	var jsonR TimeInterval //max만 사용함
	if err := c.ShouldBindJSON(&jsonR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := now.Parse(jsonR.MaxDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	btimeTime := now.With(t).BeginningOfMonth()
	bTime := btimeTime.Format(time.RFC3339)
	// bTimeDay := bTime[8:10]
	eTime := now.With(t).EndOfMonth().Format(time.RFC3339)
	eTimeDay, err := strconv.Atoi(eTime[8:10])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	query := `
	  query{
		users(where:{status:0,createdAt_gte:"` + bTime + `",createdAt_lte:"` + eTime + `"}){
		  createdAt
		},
		withdrawUsers: users(where:{status_in:[1,2],deletedAt_gte:"` + bTime + `",deletedAt_lte:"` + eTime + `"}){
		  deletedAt
		}
	  }`
	// fmt.Println(bTime)
	// fmt.Println(bTimeDay)
	// fmt.Println(eTime)
	// fmt.Println(eTimeDay)
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	_ = result
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var tt TimeTable
	tt.Columns = append(tt.Columns, "날짜/시간")
	for i := 0; i < 24; i++ {
		tt.Columns = append(tt.Columns, fmt.Sprintf("%02d", i))
	}
	for i := 0; i < eTimeDay; i++ {
		defaultTTMap := make(map[string]string)
		for i := 0; i < 24; i++ {
			defaultTTMap[fmt.Sprintf("%02d", i)] = ""
		}
		dateStr := btimeTime.AddDate(0, 0, i).Format("06-01-02")
		defaultTTMap["날짜/시간"] = dateStr
		tt.Data = append(tt.Data, defaultTTMap)
		defaultTTMap2 := make(map[string]string)
		for i := 0; i < 24; i++ {
			defaultTTMap2[fmt.Sprintf("%02d", i)] = ""
		}
		dateStr2 := btimeTime.AddDate(0, 0, i).Format("06-01-02")
		defaultTTMap2["날짜/시간"] = dateStr2
		tt.Data2 = append(tt.Data2, defaultTTMap2)
	}
	users := result["users"].([]interface{})
	withdrawUsers := result["withdrawUsers"].([]interface{})
	dataCount := 0
	data2Count := 0
	for _, v := range users {
		jsonString, _ := json.Marshal(v)
		s := CustomUser{}
		json.Unmarshal(jsonString, &s)
		// fmt.Println(s)
		matchStick := s.CreatedAt[2:10]
		matchStick2 := s.CreatedAt[11:13]
		dataCount++
		for _, v := range tt.Data {
			if v["날짜/시간"] == matchStick {
				if v[matchStick2] == "" {
					v[matchStick2] = "1"
				} else {
					oNum, err := strconv.Atoi(v[matchStick2])
					if err != nil {
						continue
					}
					v[matchStick2] = strconv.Itoa(oNum + 1)
				}
			}
		}
	}
	for _, v := range withdrawUsers {
		jsonString, _ := json.Marshal(v)
		s := CustomUser{}
		json.Unmarshal(jsonString, &s)
		// fmt.Println(s)
		matchStick := s.DeletedAt[2:10]
		matchStick2 := s.DeletedAt[11:13]
		data2Count++
		for _, v := range tt.Data2 {
			if v["날짜/시간"] == matchStick {
				if v[matchStick2] == "" {
					v[matchStick2] = "1"
				} else {
					oNum, err := strconv.Atoi(v[matchStick2])
					if err != nil {
						continue
					}
					v[matchStick2] = strconv.Itoa(oNum + 1)
				}
			}
		}

	}
	tt.DataCount = dataCount
	tt.Data2Count = data2Count

	c.JSON(http.StatusOK, tt)
}

func userList(c *gin.Context) {
	var jsonS GotchuUsers
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	whereStr := `where:{`
	if jsonS.ContainedNickname != "" {
		whereStr += `OR:[{nickName_contains:"` + jsonS.ContainedNickname + `"},{page:{nickName_contains:"` + jsonS.ContainedNickname + `"}}]`
	}
	if jsonS.ContainedEmail != "" {
		whereStr += `,email_contains:"` + jsonS.ContainedEmail + `"`
	}
	if jsonS.SubEmailKey == 1 {
		whereStr += `,subscribeEmail:true`
	} else if jsonS.SubEmailKey == 2 {
		whereStr += `,subscribeEmail:false`
	}
	if jsonS.MinDate != "" {
		whereStr += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		whereStr += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	whereStr += `,status:0}`
	query := `
	query{
		users(`
	query += whereStr
	query += `,orderBy:createdAt_DESC,first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `){
		id
		createdAt
		nickName
		email
		subscribeEmail
		page{
			nickName
		}
		cashHistories(where:{type_in:[1,2],property_in:[1,2,11]},orderBy:createdAt_DESC){
		  type
		  property
		  qty
		  oPrice
		}
	   }
	   usersConnection(`
	query += whereStr
	query += `){
			aggregate{
			  count
			}
		}
	 }`
	// fmt.Println(query) //for test
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cUsers := getPrismaUsersData(result)
	rData := makeUserListReturn(cUsers)
	count := getAggregateCount(result["usersConnection"])

	c.JSON(http.StatusOK, gin.H{"result": rData, "totalItemCount": count})
}

func getPrismaUserData(userQ map[string]interface{}) CustomUser {
	pages := userQ["user"].(interface{})
	jsonString, _ := json.Marshal(pages)
	s := CustomUser{}
	json.Unmarshal(jsonString, &s)
	return s
}

func getPrismaUsersData(userQ map[string]interface{}) []CustomUser {
	pages := userQ["users"].([]interface{})
	var fResult []CustomUser
	for _, v := range pages {
		jsonString, _ := json.Marshal(v)
		s := CustomUser{}
		json.Unmarshal(jsonString, &s)
		fResult = append(fResult, s)
	}
	return fResult
}

func getPrismaPage(pageI interface{}) CustomPage {
	pages := pageI
	jsonString, _ := json.Marshal(pages)
	s := CustomPage{}
	json.Unmarshal(jsonString, &s)
	return s
}

func getOrderHistories(queryResult map[string]interface{}) (fResult []CustomOrderHistory) {
	pages := queryResult["orderHistories"].([]interface{})
	for _, v := range pages {
		jsonString, _ := json.Marshal(v)
		s := CustomOrderHistory{}
		json.Unmarshal(jsonString, &s)
		fResult = append(fResult, s)
	}
	return fResult
}

func getGWCData(byteData []byte) (data GWCResult) {
	json.Unmarshal(byteData, &data)
	return data
}

func getYouTubePageInfoData(byteData []byte) (data YouTubePageInfo) {
	json.Unmarshal(byteData, &data)
	return data
}

func getTwitchUserInfoData(byteData []byte) (data TwitchUserInfoResult) {
	json.Unmarshal(byteData, &data)
	return data
}

func getAfreecaData(byteData []byte) (data AfreecaResult) {
	json.Unmarshal(byteData, &data)
	return data
}

func makeUserListReturn(cUsers []CustomUser) (returnStructs []UserListReturnData) {
	for _, v := range cUsers {
		s := UserListReturnData{}
		deSum, wiSum := calDepositAndWithdrawSumStruct(v.CustomCashHistories)
		s.ID = v.ID
		s.Email = v.Email
		s.CreatedAt = v.CreatedAt
		s.NickName = v.NickName
		s.PageNickName = v.CustomPage.NickName
		s.SubscribeEmail = v.SubscribeEmail
		s.DepositSum = deSum
		s.WithdrawSum = wiSum
		returnStructs = append(returnStructs, s)
	}
	return returnStructs
}
func makeDeactivateUserListReturn(cUsers []CustomUser) (returnStructs []DeactivateUserListReturnData) {
	for _, v := range cUsers {
		s := DeactivateUserListReturnData{}
		deSum, _ := calDepositAndWithdrawSumStruct(v.CustomCashHistories)
		s.ID = v.ID
		s.Email = v.Email
		s.DeletedAt = v.DeletedAt
		s.CreatedAt = v.CreatedAt
		s.UpdatedAt = v.UpdatedAt
		s.Status = v.Status
		s.NickName = v.NickName
		s.SubscribeEmail = v.SubscribeEmail
		s.DepositSum = deSum
		if v.Status == 1 {
			s.EDD = calEDD(v.DeletedAt)
		}
		returnStructs = append(returnStructs, s)
	}
	return returnStructs
}

func calEDD(updatedAt string) (edd string) {
	t, err := now.Parse(updatedAt)
	if err != nil {
		return edd
	}
	afterN := t.Add(ninetyDays)
	edd += afterN.Format(time.RFC3339)[0:10] + "\n"
	subTime := afterN.Sub(time.Now())
	subTImef := subTime.Hours() / 24
	if subTImef > 0 {
		edd += fmt.Sprint(math.Round(subTImef)) + "일 남음"
	}
	return edd
}

func getAggregateCount(pConnection interface{}) (count int) {
	jsonString, _ := json.Marshal(pConnection)
	s := PrismaConnection{}
	json.Unmarshal(jsonString, &s)
	return s.Aggregate.Count
}

func userDetail(c *gin.Context) {
	var jsonS GotchuUserDetail
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `
	query{
		user(where:{id:"` + jsonS.ID + `"}){
		  id
		  nickName
		  email
		  platform
		  page{
			pageId
		  }
		  verification{
			level
			hasEmail
			hasPin
			hasPhone
			hasBankAccount
			hasInter
		  }
		  createdAt
		  updatedAt
		}
		cashHistories(where:{user:{id:"` + jsonS.ID + `"},type_in:[1,2],property_in:[1,2,11]},orderBy:createdAt_DESC){
		  type
		  property
		  qty
		  oPrice
		  memo
		  createdAt
		}
		webCashHistories(where:{user:{id:"` + jsonS.ID + `"},type_in:[2,4]},orderBy:createdAt_DESC){
		  type
		  amount
		  memo
		  isWithdrawal
		  createdAt
		  shopPayment{
			amount
			amount_fee
			amount_without_fee
			custom_parameter
			pgcode
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error})
		return
	}
	webCashHistories := result["webCashHistories"].([]interface{})
	currentCash, settlementCash := getUserWebCash(webCashHistories)
	result["currentCash"] = currentCash
	result["settlementCash"] = settlementCash

	c.JSON(http.StatusOK, result)
}

func deactivateUserList(c *gin.Context) {
	var jsonS GotchuUsers
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `
	query{
		users(where:{nickName_contains:"`
	query += jsonS.ContainedNickname + `",email_contains:"` + jsonS.ContainedEmail + `"`
	if jsonS.SubEmailKey == 1 {
		query += `,subscribeEmail:true`
	} else if jsonS.SubEmailKey == 2 {
		query += `,subscribeEmail:false`
	}
	if jsonS.MinDate != "" {
		query += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		query += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	query += `,status_in:[1,2]},orderBy:deletedAt_DESC,first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `){
		id
		createdAt
		deletedAt
		updatedAt
		status
		nickName
		email
		subscribeEmail
		cashHistories(where:{type_in:[1,2],property_in:[1,2,11]},orderBy:createdAt_DESC){
		  type
		  property
		  qty
		  oPrice
		}
	   }
	   usersConnection(where:{nickName_contains:"`
	query += jsonS.ContainedNickname + `",email_contains:"` + jsonS.ContainedEmail + `"`
	if jsonS.SubEmailKey == 1 {
		query += `,subscribeEmail:true`
	} else if jsonS.SubEmailKey == 2 {
		query += `,subscribeEmail:false`
	}
	if jsonS.MinDate != "" {
		query += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		query += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	query += `,status_in:[1,2]}){
			aggregate{
			  count
			}
		}
	 }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cUsers := getPrismaUsersData(result)
	rData := makeDeactivateUserListReturn(cUsers)
	count := getAggregateCount(result["usersConnection"])

	c.JSON(http.StatusOK, gin.H{"result": rData, "totalItemCount": count})
}

func makeDeactivatingUser(c *gin.Context) {
	var jsonS GotchuUserDetail
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//처리과정 설명
	//해당 유저가 퀘스트등에 참여중인 상태인가? 해당 유저의 코인이 발행된 상태인가?
	query := `
	query{
		questMembersConnection(where:{user:{id:"` + jsonS.ID + `"},post:{type_in:[1,2,3],questStatus:1,isDel:false}}){
		  aggregate{
			count
		  }
		}
		user(where:{id:"` + jsonS.ID + `"}){
		  coin{
			status
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	count := getAggregateCount(result["questMembersConnection"])
	if count != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "questMembersConnection count not zero"})
		return
	}
	cUsers := getPrismaUserData(result)
	if cUsers.CustomCoin.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coin status not 1"})
		return
	}
	//계정을 탈퇴 진행으로 변경.
	status := int32(1)
	isoTime := time.Now().Format(time.RFC3339)
	updatedUser, err := pClient.UpdateUser(prisma.UserUpdateParams{
		Data: prisma.UserUpdateInput{
			Status:    &status,
			DeletedAt: &isoTime,
		},
		Where: prisma.UserWhereUniqueInput{
			ID: &jsonS.ID,
		},
	}).Exec(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//firebase 유저 정보 삭제
	dfuResult := deleteFirebaseUser(updatedUser.FirebaseUid)
	if !dfuResult {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deleteFirebaseUser fail"})
		return
	}
	// stibee 주소록에서 삭제
	stibeeReqBody := `["` + updatedUser.Email + `"]`
	lResp, err := stibeeConnector(stibeeReqBody, "DELETE")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	lBody, _ := ioutil.ReadAll(lResp.Body)
	if lResp.StatusCode != 200 {
		sendSlack(string(lBody), "stibeeConnector trouble >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
		return
	}
	sendSlack(jsonS.ID, "makeDeactivatingUser 탈퇴예정자 >> ")
	sendSlack(string(lBody), "makeDeactivatingUser 탈퇴예정자 stibee >> ")
	c.JSON(http.StatusOK, updatedUser)
}

func makeDeactivatedUser(c *gin.Context) {
	var jsonS GotchuUserDetail
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//처리과정 설명
	//해당 유저가 status==1 인지 체크
	query := `
	query{
		user(where:{id:"` + jsonS.ID + `"}){
		  status
		  firebaseUID
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cUsers := getPrismaUserData(result)
	// if cUsers.Status != 1 {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "user status not 1"})
	// 	return
	// }

	// 거래소 관련 처리들 요청
	urlParam := `uid=` + cUsers.FirebaseUID + `&userID=` + jsonS.ID
	lResp, err := gwcConnector("/10", urlParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	lBody, _ := ioutil.ReadAll(lResp.Body)
	if lResp.StatusCode != 200 {
		sendSlack(string(lBody), "gwcConnect trouble >> ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
		return
	}
	gwcResult := getGWCData(lBody)

	c.JSON(http.StatusOK, gwcResult)
}

func searchInfluPage(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyword empty"})
		return
	}

	query := `
	query{
		pages(first:100,where:{OR:[{nickName_contains:"` + keyword + `"},{pageId_contains:"` + keyword + `"},{id_contains:"` + keyword + `"},{owner:{nickName_contains:"` + keyword + `"}}]}){
	  id
	  pageId
		nickName
		owner{
		  nickName
		}
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func compareInfluPages(c *gin.Context) {
	var jsonS GotchuInfluPageManage
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.MainPageID == "" || jsonS.SubPageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MainPageID or SubPageID empty"})
		return
	}
	query := `
	query{
		mainPage: page(where:{id:"` + jsonS.MainPageID + `"}){
		  id,pageId,nickName,avatarUrl,coverUrl,description
		  badges{id,vote,badge{name}},youtube{id,channelName,userName,pageUrl}
		  twitch{id,userName,pageUrl},instagram{id,userName,pageUrl},afreecaTV{id,stationName,userName,pageUrl}
		  owner{id,nickName},comments{id},relatedReviews{id,title,review},fans{id},pageFanBoard{id}
		  createdAt,updatedAt
		}
		subPage: page(where:{id:"` + jsonS.SubPageID + `"}){
		  id,pageId,nickName,avatarUrl,coverUrl,description
		  badges{id,vote,badge{name}},youtube{id,channelName,userName,pageUrl}
		  twitch{id,userName,pageUrl},instagram{id,userName,pageUrl},afreecaTV{id,stationName,userName,pageUrl}
		  owner{id,nickName},comments{id},relatedReviews{id,title,review},fans{id},pageFanBoard{id}
		  createdAt,updatedAt
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func mergeInfluPages(c *gin.Context) {
	var jsonS GotchuInfluPageManage
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.MainPageID == "" || jsonS.SubPageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MainPageID or SubPageID empty"})
		return
	}
	fmt.Println(jsonS)
	query := `
	query{
		mainPage: page(where:{id:"` + jsonS.MainPageID + `"}){
			id,pageId,badges{id,voter{user{id},ip},badge{name}},youtube{id}
			twitch{id},instagram{id},afreecaTV{id}
		}
		subPage: page(where:{id:"` + jsonS.SubPageID + `"}){
			id,pageId,badges{id,voter{user{id},ip},badge{name}},youtube{id}
			twitch{id},instagram{id},afreecaTV{id}
			comments{id},relatedReviews{id},fans{id},pageFanBoard{id}
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	mainPage := getPrismaPage(result["mainPage"])
	subPage := getPrismaPage(result["subPage"])

	//?? 수정 전에 서브페이지와 메인페이지에 대한 정보를 저장해 놓을 것인가??
	//badges 처리 현재 뱃지 관련해서 바뀔 예정이라 해서 뱃지 처리는 걍 둠.

	//youtube twitch instagram afreecaTV 처리 + comments 처리 + relatedReviews 처리
	//하나도 처리할게 없을경우 요청할 필요가 없으므로 카운팅을 함.
	var diffCount int
	query2 := `mutation{updatePage(where:{id:"` + jsonS.MainPageID + `"},data:{`
	if mainPage.Youtube == nil && subPage.Youtube != nil {
		query2 += `youtube:{connect:{id:"` + subPage.Youtube.ID + `"}},`
		diffCount++
	}
	if mainPage.Twitch == nil && subPage.Twitch != nil {
		query2 += `twitch:{connect:{id:"` + subPage.Twitch.ID + `"}},`
		diffCount++
	}
	if mainPage.Instagram == nil && subPage.Instagram != nil {
		query2 += `instagram:{connect:{id:"` + subPage.Instagram.ID + `"}},`
		diffCount++
	}
	if mainPage.AfreecaTV == nil && subPage.AfreecaTV != nil {
		query2 += `afreecaTV:{connect:{id:"` + subPage.AfreecaTV.ID + `"}},`
		diffCount++
	}
	if len(subPage.Comments) != 0 {
		query2 += `comments:{connect:[`
		for i, ci := range subPage.Comments {
			if i == 0 {
				query2 += `{id:"` + ci.ID + `"}`
			} else {
				query2 += `,{id:"` + ci.ID + `"}`
			}
		}
		query2 += `]},`
		diffCount++
	}
	if len(subPage.RelatedReviews) != 0 {
		query2 += `relatedReviews:{connect:[`
		for i, ri := range subPage.RelatedReviews {
			if i == 0 {
				query2 += `{id:"` + ri.ID + `"}`
			} else {
				query2 += `,{id:"` + ri.ID + `"}`
			}
		}
		query2 += `]},`
		diffCount++
	}
	if len(subPage.Fans) != 0 {
		query2 += `fans:{connect:[`
		for i, ri := range subPage.Fans {
			if i == 0 {
				query2 += `{id:"` + ri.ID + `"}`
			} else {
				query2 += `,{id:"` + ri.ID + `"}`
			}
		}
		query2 += `]},`
		diffCount++
	}
	if len(subPage.PageFanBoards) != 0 {
		query2 += `pageFanBoard:{connect:[`
		for i, ri := range subPage.PageFanBoards {
			if i == 0 {
				query2 += `{id:"` + ri.ID + `"}`
			} else {
				query2 += `,{id:"` + ri.ID + `"}`
			}
		}
		query2 += `]},`
		diffCount++
	}
	query2 += `}){id}}`
	if diffCount == 0 {
		//업데이트할 거리가 없음. query2 요청 안보냄.
		fmt.Println("query2 no needs")
		fmt.Println(jsonS)
	} else {
		// fmt.Println(query2)
		variables2 := make(map[string]interface{})
		result2, err := pClient.GraphQL(ctx, query2, variables2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Println(result2)
	}
	//subPage 삭제 (현재 계획상으론 여기까지가 끝)
	query3 := `mutation{
		deletePage(where:{id:"` + jsonS.SubPageID + `"}){
		  id
		}
	  }`
	variables3 := make(map[string]interface{})
	result3, err := pClient.GraphQL(ctx, query3, variables3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println(result3)
	// fmt.Println(jsonS.BeforeData)
	pageFanCountUpdate(mainPage.PageID)
	sendSlack(result3, "Pages deleted due to page consolidation >> ")
	// sendSlack(jsonS.BeforeData, "통합전 페이지들 정보 >> ")
	c.JSON(http.StatusOK, result3)
}

func snsAcountLinkList(c *gin.Context) {
	var jsonS GotchuSNSAcountLinkList
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.First == 0 || jsonS.First > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first invalid value"})
		return
	}
	if jsonS.OrderBy == "" {
		jsonS.OrderBy = `createdAt_DESC`
	}
	whereStr := ""
	connectionWhereStr := ""
	if jsonS.ContainedNickname != "" {
		whereStr = `where:{page:{nickName_contains:"` + jsonS.ContainedNickname + `"}},`
		connectionWhereStr = `(where:{page:{nickName_contains:"` + jsonS.ContainedNickname + `"}})`
	} else if jsonS.ContainedPageID != "" {
		whereStr = `where:{page:{pageId_contains:"` + jsonS.ContainedPageID + `"}},`
		connectionWhereStr = `(where:{page:{pageId_contains:"` + jsonS.ContainedPageID + `"}})`
	}
	query := `{
		sNSAccountLinkingRequests(` + whereStr + `first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `,orderBy:` + jsonS.OrderBy + `){
		  id
		  snsType
		  urlValue
		  authenticationValue
		  isAuth
		  optionValue
		  description
		  page{
			pageId
			nickName
		  }
		  status
		  createdAt
		}
		sNSAccountLinkingRequestsConnection` + connectionWhereStr + `{
		  aggregate{
			count
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println(result)
	c.JSON(http.StatusOK, result)
}

func salp(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuSNSAcountLinkProcessing
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.SNSType > 3 || jsonS.SNSType < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SNSType invalid value"})
		return
	}
	if jsonS.UniqueValueType > 2 || jsonS.UniqueValueType < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UniqueValueType invalid value"})
		return
	}
	if jsonS.UniqueValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UniqueValue invalid value"})
		return
	}
	if jsonS.PageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PageID invalid value"})
		return
	}
	if jsonS.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalid value"})
		return
	}
	//test
	// res, err := afreecaConnector(jsonS.UniqueValue)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	// lBody, _ := ioutil.ReadAll(res.Body)
	// if res.StatusCode != 200 {
	// 	sendSlack(string(lBody), "afreecaConnector trouble >> ")
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
	// 	return
	// }
	// afreecaResult := getAfreecaData(lBody)
	// c.JSON(http.StatusOK, afreecaResult)
	// return
	//test end
	//각 sns에 따른 이미 해당 sns가 연동되어 있지 않은가 체크
	existCheckQueryWhere := ``
	if jsonS.SNSType == 0 { //instagram
		existCheckQueryWhere = `{instagram:{userId:"` + jsonS.UniqueValue + `"}}`
	} else if jsonS.SNSType == 1 { //youtube
		existCheckQueryWhere = `{youtube:{channelId:"` + jsonS.UniqueValue + `"}}`
	} else if jsonS.SNSType == 2 { //afreecaTV
		existCheckQueryWhere = `{afreecaTV:{userId:"` + jsonS.UniqueValue + `"}}`
	} else if jsonS.SNSType == 3 { //twitch
		existCheckQueryWhere = `{twitch:{userId:"` + jsonS.UniqueValue + `"}}`
	}
	existCheckQuery := `{
		pages(where:` + existCheckQueryWhere + `){
		  pageId
		}
	  }`
	ecVariables := make(map[string]interface{})
	ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(ecResult["pages"].([]interface{})) != 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "UniqueValue already exist", "hint": ecResult})
		return
	}

	//각 sns에 따른 정보 가져오기
	//각 sns에 따른 정보 유효성 체크(필수 데이터 존재 유무 체크)
	// var result map[string]interface{}
	var updateQuery string
	//SNSType에 따른 분기
	if jsonS.SNSType == 0 { //instagram
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "인스타그램은 현재 자동연동 불가입니다."})
		// return
		insResult, err := instagramConnector(jsonS.UniqueValue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// fmt.Println(insResult)
		if insResult.ID != "" {
			user := insResult //이거 굳이 유지할 필요 없지만 아래 코드 고치기 귀찮아서 걍 이렇게 놔둠.

			updateQuery = `mutation{
			updatePage(where:{pageId:"` + jsonS.PageID + `"},data:{instagram:{create:{userId:"` + user.UserName + `",userNo:"` + user.ID +
				`",userName:"` + removeSC(user.FullName) + `",postCount:"` + strconv.Itoa(user.Post.Count) + `",followerCount:"` + strconv.Itoa(user.Follower.Count) +
				`",followingCount:"` + strconv.Itoa(user.Following.Count) + `",avatarUrl:"` + user.ProfilePicURL + `",description:"` + removeSC(user.Biography) +
				`",pageUrl:"https://www.instagram.com/` + user.UserName + `"}}}){
			  id
			}
		  }`
			// fmt.Println(updateQuery)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "instagramConnector result no has ID"})
			return
		}
	} else if jsonS.SNSType == 1 { //youtube
		urlParam := `part=id,snippet,statistics,brandingSettings&id=` + jsonS.UniqueValue + `&key=` + youtubeAPIKey
		lResp, err := youTubeChannelInfoConnector(urlParam)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		lBody, _ := ioutil.ReadAll(lResp.Body)
		if lResp.StatusCode != 200 {
			sendSlack(string(lBody), "youTubeChannelInfoConnector trouble >> ")
			c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
			return
		}
		// fmt.Println(string(lBody))
		yPIResult := getYouTubePageInfoData(lBody)
		if len(yPIResult.Items) != 0 {
			item := yPIResult.Items[0]

			// fmt.Println(item)
			// fmt.Println(if item["cat"].(string) { item["cat"].(string) } else { "empty" })
			updateQuery = `mutation{
			updatePage(where:{pageId:"` + jsonS.PageID + `"},data:{youtube:{create:{channelId:"` + item.ID + `",channelType:"` + item.Kind +
				`",channelName:"` + item.Snippet.Title + `",userName:"",bannerUrl:"` + item.BrandingSettings.Image.BannerImageURL + `",videoCount:"` + item.Statistics.VideoCount +
				`",subscriberCount:"` + item.Statistics.SubscriberCount + `",videoViewCount:"` + item.Statistics.ViewCount +
				`",thumbnailUrl:"` + item.Snippet.Thumbnails.High.URL +
				`",description:"` + removeSC(item.Snippet.Description) + `",publishedAt:"` + item.Snippet.PublishedAt + `",country:"` + item.Snippet.Country +
				`",pageUrl:"https://www.youtube.com/channel/` + item.ID + `"}}}){
			  id
			}
		  }`
			// fmt.Println(updateQuery)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "youTubeChannelInfoConnector no result"})
			return
		}
		//   ` + jsonS.SubPageID + `
		// fmt.Println("Id :", result["id"],
		// 	"\nName :", result["name"],
		// 	"\nDepartment :", result["department"],
		// 	"\nDesignation :", result["designation"])
	} else if jsonS.SNSType == 2 { //afreecaTV
		res, err := afreecaConnector(jsonS.UniqueValue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		lBody, _ := ioutil.ReadAll(res.Body)
		if res.StatusCode != 200 {
			sendSlack(string(lBody), "afreecaConnector trouble >> ")
			c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
			return
		}
		afreecaResult := getAfreecaData(lBody)
		updateQuery = `mutation{
			updatePage(where:{pageId:"` + jsonS.PageID + `"},data:{afreecaTV:{create:{userId:"` + afreecaResult.Station.UserID + `",userName:"` + afreecaResult.Station.UserNick +
			`",stationNo:"` + strconv.Itoa(afreecaResult.Station.StationNo) + `",stationName:"` + afreecaResult.Station.StationName + `",stationTitle:"` + afreecaResult.Station.StationTitle +
			`",avatarUrl:"` + afreecaResult.ProfileImage + `",followerCount:"` + strconv.Itoa(afreecaResult.Station.Upd.FanCnt) + `",description:"` + removeSC(afreecaResult.Station.Display.ProfileText) + `",viewCount:"` + strconv.Itoa(afreecaResult.Station.Upd.TotalViewCnt) + `",visitCount:"` + strconv.Itoa(afreecaResult.Station.Upd.TotalVisitCnt) + `",fanCount:"` + strconv.Itoa(afreecaResult.Station.Upd.FanCnt) +
			`",pageUrl:"https://bj.afreecatv.com/` + afreecaResult.Station.UserID + `"}}}){
			  id
			}
		  }`
	} else if jsonS.SNSType == 3 { //twitch
		res, err := twitchUserInfoConnector(jsonS.UniqueValue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		lBody, _ := ioutil.ReadAll(res.Body)
		if res.StatusCode != 200 {
			sendSlack(string(lBody), "twitchUserInfoConnector trouble >> ")
			c.JSON(http.StatusInternalServerError, gin.H{"error": string(lBody)})
			return
		}
		tUIResult := getTwitchUserInfoData(lBody)
		if len(tUIResult.Data) != 0 {
			user := tUIResult.Data[0]
			followerCount, followingCount := twitchUserFollowConnector(user.ID)

			updateQuery = `mutation{
				updatePage(where:{pageId:"` + jsonS.PageID + `"},data:{twitch:{create:{userId:"` + user.Login + `",userNo:"` + user.ID +
				`",userName:"` + user.DisplayName + `",avatarUrl:"` + user.ProfileImageURL + `",channelViewCount:"` + strconv.Itoa(user.ViewCount) +
				`",followerCount:"` + fmt.Sprint(followerCount) + `",followingCount:"` + fmt.Sprint(followingCount) +
				`",description:"` + removeSC(user.Description) +
				`",pageUrl:"https://www.twitch.tv/` + user.Login + `"}}}){
				  id
				}
			  }`
		}
	}

	//각 sns에 따른 연동 처리(페이지 업데이트 )
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, updateQuery, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//완료 확인 후 연동처리 신청 데이터 갱신 및 응답
	updateSALRQuery := `mutation{
		updateSNSAccountLinkingRequest(where:{id:"` + jsonS.ID + `"},data:{status:1,isAuth:true}){
		  id
		  page{nickName,pageId}
		}
	  }`
	SALRvariable := make(map[string]interface{})
	SALRResult, err := pClient.GraphQL(ctx, updateSALRQuery, SALRvariable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sendSlack(SALRResult, "SNS 연동 처리 완료 >> ")
	c.JSON(http.StatusOK, result)
}

func refuseSNSLink(c *gin.Context) {
	var jsonS GotchuSNSAcountLinkProcessing
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalid value"})
		return
	}
	updateSALRQuery := `mutation{
		updateSNSAccountLinkingRequest(where:{id:"` + jsonS.ID + `"},data:{status:2}){
		  id
		}
	  }`
	SALRvariable := make(map[string]interface{})
	SALRResult, err := pClient.GraphQL(ctx, updateSALRQuery, SALRvariable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, SALRResult)
}

func reviewList(c *gin.Context) {
	var jsonS GotchuReviewList
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.First == 0 || jsonS.First > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first invalid value"})
		return
	}
	if jsonS.OrderBy == "" {
		jsonS.OrderBy = `createdAt_DESC`
	}
	whereStr := `where:{owner:{nickName_contains:"` + jsonS.ContainedNickname + `"}`
	connectionWhereStr := `(where:{owner:{nickName_contains:"` + jsonS.ContainedNickname + `"}`
	if jsonS.MinDate != "" {
		whereStr += `,createdAt_gte:"` + jsonS.MinDate + `"`
		connectionWhereStr += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		whereStr += `,createdAt_lte:"` + jsonS.MaxDate + `"`
		connectionWhereStr += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	whereStr += `},`
	connectionWhereStr += `})`
	query := `{
		reviewContentReviews(` + whereStr + `first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `,orderBy:` + jsonS.OrderBy + `){
			id
			reviewContent{
			  previewImageUrl
			}
			owner{
			  nickName
			}
			review
			likeCount
			commentCount
			isDel
			createdAt
		}
		reviewContentReviewsConnection` + connectionWhereStr + `{
		  aggregate{
			count
		  }
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println(result)
	c.JSON(http.StatusOK, result)
}

func deleteReview(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuDeleteReview
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalid value"})
		return
	}
	//test
	//test end
	existCheckQuery := `{
		reviewContentReview(where:{id:"` + jsonS.ID + `"}){
		  isDel
		}
	  }`
	ecVariables := make(map[string]interface{})
	ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ecResult["reviewContentReview"] == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not found review"})
		return
	}
	isDel := (ecResult["reviewContentReview"].(map[string]interface{}))["isDel"].(bool)
	if isDel {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "this review already delete"})
		return
	}
	updateRCRQuery := `mutation{
		updateReviewContentReview(where:{id:"` + jsonS.ID + `"},data:{isDel:true}){
		  id
		  reviewContent{
		    id
		  }
		}
	  }`
	RCRvariable := make(map[string]interface{})
	RCRResult, err := pClient.GraphQL(ctx, updateRCRQuery, RCRvariable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	reviewContentCountUpdate(((RCRResult["updateReviewContentReview"].(map[string]interface{}))["reviewContent"].(map[string]interface{}))["id"].(string)) //reviewContent의 rcr갯수 갱신필요
	c.JSON(http.StatusOK, RCRResult)
	return
}

func reviewContentCountUpdate(contentID string) {
	query := `{
		reviewContentReviewsConnection(where:{isDel:false,reviewContent:{id:"` + contentID + `"}}){
		  aggregate{
			count
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		sendSlack(err.Error(), "reviewContentCountUpdate reviewContentReviewsConnection err >> ")
		return
	}
	rcrCount := result["reviewContentReviewsConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(float64)
	query2 := `mutation{
		updateReviewContent(where:{id:"` + contentID + `"},data:{reviewCount:` + strconv.Itoa(int(rcrCount)) + `}){
		  id
		}
	  }`
	variables2 := make(map[string]interface{})
	_, err2 := pClient.GraphQL(ctx, query2, variables2)
	if err2 != nil {
		sendSlack(err2.Error(), "reviewContentCountUpdate updateReviewContent err >> ")
		return
	}
}

func pageFanCountUpdate(pageID string) {
	query := `{
		pageFansConnection(where:{page:{pageId:"` + pageID + `"}}){
		  aggregate{
			count
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		sendSlack(err.Error(), "pageFanCountUpdate pageFansConnection err >> ")
		return
	}
	fanCount := result["pageFansConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(float64)
	query2 := `mutation{
		updatePage(where:{pageId:"` + pageID + `"},data:{fanCount:` + strconv.Itoa(int(fanCount)) + `}){
		  id
		}
	  }`
	variables2 := make(map[string]interface{})
	_, err2 := pClient.GraphQL(ctx, query2, variables2)
	if err2 != nil {
		sendSlack(err2.Error(), "pageFanCountUpdate updatePage err >> ")
		return
	}
}

func ticketInfo1(c *gin.Context) {
	query := `{
		pageShopOrders(where:{status_in:[2,3]},first:1000,skip:0){
		  productPrice
		  qty
		}
		pageShopOrdersConnection(where:{status_in:[2,3]}){
		  aggregate{
			count
		  }
		}
		pagesConnection(where:{pageShopOrders_some:{status_in:[2,3]}}){
		  aggregate{
			count
		  }
		}
		usersConnection(where:{pageShopOrders_some:{status_in:[2,3]}}){
		  aggregate{
			count
		  }
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println(result)
	// fmt.Println(result["pageShopOrdersConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(int))
	pageShopOrders := result["pageShopOrders"].([]interface{})
	var totalTicketSales float64
	for _, v := range pageShopOrders {
		totalTicketSales += v.(map[string]interface{})["productPrice"].(float64) * v.(map[string]interface{})["qty"].(float64)
	}
	// fmt.Println(totalTicketSales)
	numberOfTicketsPurchased := result["pageShopOrdersConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(float64)
	if numberOfTicketsPurchased > 1000 {
		loopTime := int(numberOfTicketsPurchased / 1000)
		for i := 0; i < loopTime; i++ {
			lquery := `{
				pageShopOrders(where:{status_in:[2,3]},first:1000,skip:` + strconv.Itoa(1000*(i+1)) + `){
				  productPrice
				  qty
				}
			  }`
			lvariables := make(map[string]interface{})
			lresult, err := pClient.GraphQL(ctx, lquery, lvariables)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			lpageShopOrders := lresult["pageShopOrders"].([]interface{})
			for _, v := range lpageShopOrders {
				totalTicketSales += v.(map[string]interface{})["productPrice"].(float64) * v.(map[string]interface{})["qty"].(float64)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"totalTicketSales": totalTicketSales,
		"haveSoldTicketsCount":        result["pagesConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(float64),
		"numberOfTicketsPurchased":    numberOfTicketsPurchased,
		"numberOfUserTicketPurchased": result["usersConnection"].(map[string]interface{})["aggregate"].(map[string]interface{})["count"].(float64)})
	return
}

func ticketUserList(c *gin.Context) {
	var jsonS GotchuTicketUserList
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.First == 0 || jsonS.First > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first invalid value"})
		return
	}
	if jsonS.OrderBy == "" {
		jsonS.OrderBy = `createdAt_DESC`
	}
	whereStr := `where:{pageShopOrders_some:{status_in:[2,3]},nickName_contains:"` + jsonS.ContainedNickname + `"`
	connectionWhereStr := `(where:{pageShopOrders_some:{status_in:[2,3]},nickName_contains:"` + jsonS.ContainedNickname + `"`
	whereStr += `},`
	connectionWhereStr += `})`
	query := `{
		pages(` + whereStr + `first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `,orderBy:` + jsonS.OrderBy + `){
			id
			pageId
			nickName
			pageShopOrders(where:{status_in:[2,3]},first:1000,skip:0){
			  productPrice
			  qty
			}
			useShop
		}
		pagesConnection` + connectionWhereStr + `{
		  aggregate{
			count
		  }
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println(result)
	for i, v := range result["pages"].([]interface{}) {
		pageShopOrders := v.(map[string]interface{})["pageShopOrders"].([]interface{})
		pageShopOrdersLen := len(pageShopOrders)
		var totalTicketSales float64
		for _, v := range pageShopOrders {
			totalTicketSales += v.(map[string]interface{})["productPrice"].(float64) * v.(map[string]interface{})["qty"].(float64)
		}
		result["pages"].([]interface{})[i].(map[string]interface{})["totalSaleAmount"] = totalTicketSales
		result["pages"].([]interface{})[i].(map[string]interface{})["totalSaleTimes"] = pageShopOrdersLen
		delete(result["pages"].([]interface{})[i].(map[string]interface{}), "pageShopOrders")
	}
	c.JSON(http.StatusOK, result)
}

func getUserWebCash(webCashHistories []interface{}) (currentCash float64, settlementCash float64) {
	for _, v := range webCashHistories {
		vv := v.(map[string]interface{})
		if vv["type"].(float64) == 4 {
			currentCash += vv["amount"].(float64)
			if vv["isWithdrawal"].(bool) == true {
				settlementCash -= vv["amount"].(float64)
			}
		} else {
			currentCash += vv["amount"].(float64)
		}
	}
	return currentCash, settlementCash
}

func ticketUserDetail(c *gin.Context) {
	var jsonS GotchuTicketUserDetail
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.PageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pageId empty"})
		return
	}
	query := `{
		page(where:{pageId:"` + jsonS.PageID + `"}){
		  owner{
			nickName
			webCashHistories(where:{user:{page:{pageId:"` + jsonS.PageID + `"}},type_in:[2,4]},orderBy:createdAt_DESC){
			  type
			  isWithdrawal
			  amount
			}
		  }
		  useShop
		  useShopM
		  pageId
		  shopProducts{
			id
			name
			price
			stockQty
			description
			slotId
			shortID
			status
			createdAt
			updatedAt
		  }
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	webCashHistories := result["page"].(map[string]interface{})["owner"].(map[string]interface{})["webCashHistories"].([]interface{})
	currentCash, settlementCash := getUserWebCash(webCashHistories)
	result["page"].(map[string]interface{})["owner"].(map[string]interface{})["currentCash"] = currentCash
	result["page"].(map[string]interface{})["owner"].(map[string]interface{})["settlementCash"] = settlementCash
	delete(result["page"].(map[string]interface{})["owner"].(map[string]interface{}), "webCashHistories")

	c.JSON(http.StatusOK, result)
	return
}

func pageShopOrderList(c *gin.Context) {
	var jsonS GotchuPageShopOrderList
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.First == 0 || jsonS.First > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first invalid value"})
		return
	}
	if jsonS.OrderBy == "" {
		jsonS.OrderBy = `createdAt_DESC`
	}
	whereStr := ``
	connectionWhereStr := ``
	// where:{page:{pageId:"jhmun2",nickName_contains:"문"},user:{nickName_contains:"문"}}
	if jsonS.PageID != "" {
		if jsonS.Contents != "" {
			if jsonS.Selected == 0 {
				whereStr = `where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `",nickName_contains:"` + jsonS.Contents + `"}`
				connectionWhereStr = `(where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `",nickName_contains:"` + jsonS.Contents + `"}`
			} else {
				whereStr = `where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `"},user:{nickName_contains:"` + jsonS.Contents + `"}`
				connectionWhereStr = `(where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `"},user:{nickName_contains:"` + jsonS.Contents + `"}`
			}
		} else {
			whereStr = `where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `"}`
			connectionWhereStr = `(where:{status_in:[2,3],page:{pageId:"` + jsonS.PageID + `"}`
		}
	} else {
		if jsonS.Contents != "" {
			if jsonS.Selected == 0 {
				whereStr = `where:{status_in:[2,3],page:{nickName_contains:"` + jsonS.Contents + `"}`
				connectionWhereStr = `(where:{status_in:[2,3],page:{nickName_contains:"` + jsonS.Contents + `"}`
			} else {
				whereStr = `where:{status_in:[2,3],user:{nickName_contains:"` + jsonS.Contents + `"}`
				connectionWhereStr = `(where:{status_in:[2,3],user:{nickName_contains:"` + jsonS.Contents + `"}`
			}
		} else {
			whereStr = `where:{status_in:[2,3]`
			connectionWhereStr = `(where:{status_in:[2,3]`
		}
	}
	if jsonS.MinDate != "" {
		whereStr += `,createdAt_gte:"` + jsonS.MinDate + `"`
		connectionWhereStr += `,createdAt_gte:"` + jsonS.MinDate + `"`
	}
	if jsonS.MaxDate != "" {
		whereStr += `,createdAt_lte:"` + jsonS.MaxDate + `"`
		connectionWhereStr += `,createdAt_lte:"` + jsonS.MaxDate + `"`
	}
	whereStr += `},`
	connectionWhereStr += `})`
	query := `{
		pageShopOrders(` + whereStr + `first:` + strconv.Itoa(jsonS.First) + `,skip:` + strconv.Itoa(jsonS.Skip) + `,orderBy:` + jsonS.OrderBy + `){
			id
			status
			page{
			  nickName
			}
			productId
			productName
			productDescription
			productPrice
			qty
			comment
			user{
			  nickName
			  page{
				nickName
			  }
			}
			confirmation{
			  id
			  imageUrl
			  videoUrl
			  memo
			}
			payment{
			  custom_parameter
			  pgcode
			}
			confirmationAt
			confirmAt
			paymentAt
			createdAt
			updatedAt
		}
		pageShopOrdersConnection` + connectionWhereStr + `{
		  aggregate{
			count
		  }
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
	return
}

func updateUseShopM(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuUpdateUseShopM
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.PageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PageID empty"})
		return
	}
	//test
	//test end
	existCheckQuery := `{
		page(where:{pageId:"` + jsonS.PageID + `"}){
			useShopM
		}
	  }`
	ecVariables := make(map[string]interface{})
	ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ecResult["page"] == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not found Page"})
		return
	}
	currentUseShopM := (ecResult["page"].(map[string]interface{}))["useShopM"].(bool)
	useShopMStr := "true"
	if currentUseShopM {
		useShopMStr = "false"
	}
	updatePQuery := `mutation{
		updatePage(where:{pageId:"` + jsonS.PageID + `"},data:{useShopM:` + useShopMStr + `}){
			useShopM
		}
	  }`
	Pvariable := make(map[string]interface{})
	PResult, err := pClient.GraphQL(ctx, updatePQuery, Pvariable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PResult)
	return
}

func uploadImage(c *gin.Context) {
	file, fileHeader, _ := c.Request.FormFile("file")
	splitHeaderName := strings.Split(fileHeader.Filename, ".")
	if len(splitHeaderName) > 2 || len(splitHeaderName) <= 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid fileName"})
		return
	}
	filename := splitHeaderName[0] + "_" + strconv.FormatInt(time.Now().Unix(), 10) + "." + splitHeaderName[1]
	imageURL := uploadFile(filename, file)
	if imageURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload fail"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"imageUrl": imageURL})
	return
}

func upsertBanner(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuWebBanner
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.BannerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BannerName empty"})
		return
	}
	if jsonS.SortNum == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SortNum invalid value"})
		return
	}
	if jsonS.Status == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status invalid value"})
		return
	}
	//test
	//test end
	dataQuery := `sortNum:` + strconv.Itoa(jsonS.SortNum) + `,status:` + strconv.Itoa(jsonS.Status) + `,BannerName:"` + jsonS.BannerName + `"`
	if jsonS.ImageURLPC != "" {
		dataQuery += `,ImageUrl_PC:"` + jsonS.ImageURLPC + `"`
	}
	if jsonS.ImageURLM != "" {
		dataQuery += `,ImageUrl_M:"` + jsonS.ImageURLM + `"`
	}
	if jsonS.Description != "" {
		dataQuery += `,description:"` + jsonS.Description + `"`
	}
	if jsonS.LinkURL != "" {
		dataQuery += `,linkURL:"` + jsonS.LinkURL + `"`
	}
	if jsonS.Option != "" {
		dataQuery += `,option:"` + jsonS.Option + `"`
	}

	query := `mutation{
		upsertGotchuWebBanner(where:{BannerName:"` + jsonS.BannerNameO + `"},
		  update:{` + dataQuery + `},
		  create:{` + dataQuery + `}){
		  id
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
	return
}

func webBannerList(c *gin.Context) {
	query := `{
		gotchuWebBanners(where:{status_in:[1,2]},orderBy:sortNum_ASC){
		  id
		  BannerName
		  ImageUrl_PC
		  ImageUrl_M
		  description
		  linkURL
		  option
		  sortNum
		  status
		  createdAt
		}
		gotchuWebBannersConnection(where:{status_in:[1,2]}){
		  aggregate{
			count
		  }
		}
	  }`
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func upsertPSOC(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuPageShopOrderConfirm
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.PageShopOrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PageShopOrderID empty"})
		return
	}
	if jsonS.Option == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Option empty"})
		return
	}
	//test
	//test end
	dataStr := ``
	switch jsonS.Option {
	case "ci":
		dataStr = `{confirmation:{create:{imageUrl:"` + jsonS.UploadImageURL + `"}}}`
	case "cv":
		dataStr = `{confirmation:{create:{videoUrl:"` + jsonS.VideoURL + `"}}}`
	case "ct":
		dataStr = `{confirmation:{create:{memo:"""` + jsonS.Memo + `"""}}}`
	case "ui":
		dataStr = `{confirmation:{update:{imageUrl:"` + jsonS.UploadImageURL + `"}}}`
	case "uv":
		dataStr = `{confirmation:{update:{videoUrl:"` + jsonS.VideoURL + `"}}}`
	case "ut":
		dataStr = `{confirmation:{update:{memo:"""` + jsonS.Memo + `"""}}}`
	case "di":
		dataStr = `{confirmation:{update:{imageUrl:""}}}`
	case "dv":
		dataStr = `{confirmation:{update:{videoUrl:""}}}`
	case "dt":
		dataStr = `{confirmation:{update:{memo:""}}}`
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Option invalid value"})
		return
	}
	query := `mutation{
		updatePageShopOrder(where:{id:"` + jsonS.PageShopOrderID + `"},data:` + dataStr + `){
			id
			status
			page{
			  nickName
			}
			productId
			productName
			productDescription
			productPrice
			qty
			comment
			user{
			  nickName
			  page{
				nickName
			  }
			}
			confirmation{
			  id
			  imageUrl
			  videoUrl
			  memo
			}
			payment{
			  custom_parameter
			  pgcode
			}
			confirmationAt
			confirmAt
			paymentAt
			createdAt
			updatedAt
		}
	  }`
	// fmt.Println(query)
	variables := make(map[string]interface{})
	result, err := pClient.GraphQL(ctx, query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
	return
}

func adminWebCashChange(c *gin.Context) {
	//전달 받은 데이터 가공, 유효 체크
	var jsonS GotchuWebCashUpsert
	if err := c.ShouldBindJSON(&jsonS); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if jsonS.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID empty"})
		return
	}
	if jsonS.Value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value invalid"})
		return
	} else if jsonS.Value > 100000000 || jsonS.Value < -100000000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value invalid"})
		return
	}
	if jsonS.Memo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Memo empty"})
		return
	}
	//test
	//test end
	existCheckQuery := `{
		user(where:{id:"` + jsonS.ID + `"}){
			id
			firebaseUID
			email
			nickName
			page{
			  nickName
			}
		}
	  }`
	ecVariables := make(map[string]interface{})
	ecResult, err := pClient.GraphQL(ctx, existCheckQuery, ecVariables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	email := ecResult["user"].(map[string]interface{})["email"].(string)
	firebaseUID := ecResult["user"].(map[string]interface{})["firebaseUID"].(string)
	nickName := ecResult["user"].(map[string]interface{})["nickName"].(string)
	pageNickName := ecResult["user"].(map[string]interface{})["page"].(map[string]interface{})["nickName"].(string)
	userInfoStr := "이메일 : " + email + " ,firebaseUID : " + firebaseUID + " ,앱 닉네임 : " + nickName + " ,웹 닉네임 : " + pageNickName +
		" ,증감량 : " + strconv.Itoa(jsonS.Value) + " ,메모 : " + jsonS.Memo
	wchType := "2" //웹캐쉬히스토리의 타입 값. 양값이면 2 음값이면 4
	if jsonS.Value < 0 {
		wchType = "4"
	}
	updatePQuery := `mutation{
		createWebCashHistory(data:{user:{connect:{id:"` + jsonS.ID + `"}},type:` + wchType + `,amount:` + strconv.Itoa(jsonS.Value) + `,memo:"""` + jsonS.Memo + `"""}){
		  id
		}
	  }`
	Pvariable := make(map[string]interface{})
	PResult, err := pClient.GraphQL(ctx, updatePQuery, Pvariable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = PResult
	sendSlack(userInfoStr, "어드민 웹 캐시 증감 >> ")

	c.JSON(http.StatusOK, gin.H{"success": userInfoStr})
	return
}
