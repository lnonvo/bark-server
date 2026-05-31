package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mritd/logger"
)

var (
	geminiKey   string
	geminiModel string
)

func SetGeminiConfig(key, model string) {
	geminiKey = key
	geminiModel = model
	if key != "" {
		logger.Infof("Gemini verification code analysis enabled, model: %s", model)
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// analyzeNotificationGroup calls the Gemini API to categorize the notification
// into one of the four quadrants (Eisenhower Matrix). Returns the group name
// if successful, or empty string on any failure.
func analyzeNotificationGroup(title, body string) string {
	if geminiKey == "" || (title == "" && body == "") {
		return ""
	}

	content := ""
	if title != "" && body != "" {
		content = fmt.Sprintf("标题: %s\n内容: %s", title, body)
	} else if title != "" {
		content = fmt.Sprintf("标题: %s", title)
	} else {
		content = fmt.Sprintf("内容: %s", body)
	}

	prompt := fmt.Sprintf(
		"你是一个通知消息分类助手。请根据四象限工作法（艾森豪威尔矩阵）对以下通知消息进行分组。\n\n"+
			"四个分组如下：\n"+
			"- 重要紧急：需要立即处理的重要事项，如安全警报、服务宕机、紧急会议通知、验证码\n"+
			"- 重要不紧急：重要但可以稍后处理的事项，如周报提醒、项目进度更新、账单通知、系统更新\n"+
			"- 紧急不重要：紧急但不太重要的事项，如促销活动即将结束、外卖已送达、快递已签收\n"+
			"- 不重要不紧急：既不紧急也不重要的事项，如广告推送、新闻资讯、推荐内容、营销消息\n\n"+
			"示例：\n"+
			"- 「您的账户存在异常登录」→ 重要紧急\n"+
			"- 「您的验证码是 836201」→ 重要紧急\n"+
			"- 「服务器 CPU 使用率超过 95%%」→ 重要紧急\n"+
			"- 「明天下午3点部门会议」→ 重要不紧急\n"+
			"- 「您的月度账单已生成」→ 重要不紧急\n"+
			"- 「系统将于今晚维护升级」→ 重要不紧急\n"+
			"- 「您的外卖已送达」→ 紧急不重要\n"+
			"- 「限时优惠还剩2小时」→ 紧急不重要\n"+
			"- 「您的快递已被签收」→ 紧急不重要\n"+
			"- 「为您推荐今日热门文章」→ 不重要不紧急\n"+
			"- 「双十一大促即将开始」→ 不重要不紧急\n"+
			"- 「新用户专享优惠券」→ 不重要不紧急\n\n"+
			"请只回复分组名称（重要紧急、重要不紧急、紧急不重要、不重要不紧急），不要回复其他任何内容。\n\n"+
			"通知消息：\n%s", content)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Errorf("Gemini: failed to marshal group request: %v", err)
		return ""
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		logger.Errorf("Gemini: failed to create group request: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("Gemini: group request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Errorf("Gemini: group API returned status %d: %s", resp.StatusCode, string(respBody))
		return ""
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		logger.Errorf("Gemini: failed to decode group response: %v", err)
		return ""
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return ""
	}

	group := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)

	// Validate the response is one of the four quadrants
	validGroups := map[string]bool{
		"重要紧急":   true,
		"重要不紧急":  true,
		"紧急不重要":  true,
		"不重要不紧急": true,
	}
	if !validGroups[group] {
		logger.Warnf("Gemini: unexpected group response: %s", group)
		return ""
	}

	logger.Infof("Gemini: notification categorized as [%s]", group)
	return group
}

// analyzeVerificationCode calls the Gemini API to extract a verification code
// from the given text. Returns the code if found, or empty string on any failure.
// Failures are logged and silently ignored so the push flow is never blocked.
func analyzeVerificationCode(body string) string {
	if geminiKey == "" || body == "" {
		return ""
	}

	prompt := fmt.Sprintf(
		"Extract the verification code from the following message. "+
			"Reply with ONLY the verification code itself (digits/letters), nothing else. "+
			"If there is no verification code, reply with exactly \"NONE\".\n\n"+
			"Message: %s", body)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Errorf("Gemini: failed to marshal request: %v", err)
		return ""
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		logger.Errorf("Gemini: failed to create request: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("Gemini: request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Errorf("Gemini: API returned status %d: %s", resp.StatusCode, string(respBody))
		return ""
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		logger.Errorf("Gemini: failed to decode response: %v", err)
		return ""
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return ""
	}

	code := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)

	if code == "" || strings.EqualFold(code, "NONE") {
		return ""
	}

	logger.Infof("Gemini: extracted verification code from message")
	return code
}
