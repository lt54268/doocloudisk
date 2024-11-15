package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type UserResp struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	User User   `json:"data"`
}
type User struct {
	Userid     int      `json:"userid"`
	Identity   []string `json:"identity"`   // Assuming identity is a slice of strings
	Department []string `json:"department"` // Assuming department is a slice of strings
	Az         string   `json:"az"`
	Email      string   `json:"email"`
	Nickname   string   `json:"nickname"`
	Userimg    string   `json:"userimg"`
	LoginNum   int      `json:"login_num"`
	Changepass int      `json:"changepass"`
	LastIp     string   `json:"last_ip"`
	LastAt     string   `json:"last_at"`
	LineIp     string   `json:"line_ip"`
	LineAt     string   `json:"line_at"`
	CreatedIp  string   `json:"created_ip"`
}

func GetUserInfo(token []byte) (*User, error) {
	c, err := client.NewClient()
	if err != nil {
		return nil, err
	}
	req := &protocol.Request{}
	res := &protocol.Response{}
	req.SetMethod(consts.MethodGet)
	req.Header.Set("TOKEN", string(token))
	ip := os.Getenv("NGINX_URL")
	if ip == "" {
		ip = "nginx"
	}
	req.SetRequestURI(fmt.Sprintf("http://%s/api/users/info", ip))
	err = c.Do(context.Background(), req, res)
	if err != nil {
		return nil, err
	}
	u := new(UserResp)
	err = json.Unmarshal(res.Body(), u)

	// fmt.Printf("%v\n", string(res.Body()))
	return &u.User, nil
}
