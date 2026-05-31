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
		logger.Infof("Gemini notification analysis enabled, model: %s", model)
	}
}

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	GenerationConfig *generationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
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

// AnalysisResult holds the combined result of Gemini notification analysis.
type AnalysisResult struct {
	Code  string `json:"code"`
	Group string `json:"group"`
}

// analyzeNotification calls the Gemini API once to extract a verification code
// and categorize the notification by the Eisenhower Matrix (四象限工作法).
// needCode/needGroup control which analyses are requested.
// Failures are logged and silently ignored so the push flow is never blocked.
func analyzeNotification(title, body string, needCode, needGroup bool) AnalysisResult {
	if geminiKey == "" || (!needCode && !needGroup) {
		return AnalysisResult{}
	}
	if title == "" && body == "" {
		return AnalysisResult{}
	}

	content := ""
	if title != "" && body != "" {
		content = fmt.Sprintf("标题: %s\n内容: %s", title, body)
	} else if title != "" {
		content = fmt.Sprintf("标题: %s", title)
	} else {
		content = fmt.Sprintf("内容: %s", body)
	}

	// Build task instructions based on what's needed
	var tasks []string

	if needCode {
		tasks = append(tasks,
			"1. 提取验证码：从消息中提取验证码（数字或字母组合）。如果没有验证码，code 字段返回空字符串。")
	}
	if needGroup {
		tasks = append(tasks,
			fmt.Sprintf("%d. 通知分组：根据四象限工作法（艾森豪威尔矩阵）对通知进行分组。\n"+
				"   四个分组如下：\n"+
				"   - 重要紧急：需要立即处理的重要事项，如安全警报、服务宕机、紧急会议通知、验证码\n"+
				"   - 重要不紧急：重要但可以稍后处理的事项，如周报提醒、项目进度更新、账单通知、系统更新\n"+
				"   - 紧急不重要：紧急但不太重要的事项，如促销活动即将结束、外卖已送达、快递已签收\n"+
				"   - 不重要不紧急：既不紧急也不重要的事项，如广告推送、新闻资讯、推荐内容、营销消息\n\n"+
				"   示例：\n"+
				"   - 「您的账户存在异常登录」→ 重要紧急\n"+
				"   - 「您的验证码是 836201」→ 重要紧急\n"+
				"   - 「服务器 CPU 使用率超过 95%%」→ 重要紧急\n"+
				"   - 「明天下午3点部门会议」→ 重要不紧急\n"+
				"   - 「您的月度账单已生成」→ 重要不紧急\n"+
				"   - 「系统将于今晚维护升级」→ 重要不紧急\n"+
				"   - 「您的外卖已送达」→ 紧急不重要\n"+
				"   - 「限时优惠还剩2小时」→ 紧急不重要\n"+
				"   - 「您的快递已被签收」→ 紧急不重要\n"+
				"   - 「为您推荐今日热门文章」→ 不重要不紧急\n"+
				"   - 「双十一大促即将开始」→ 不重要不紧急\n"+
				"   - 「新用户专享优惠券」→ 不重要不紧急\n\n"+
				"   group 字段只能是以下四个值之一：重要紧急、重要不紧急、紧急不重要、不重要不紧急",
				len(tasks)+1))
	}

	prompt := fmt.Sprintf(
		"你是一个通知消息分析助手。请对以下通知消息完成指定任务，以 JSON 格式返回结果。\n\n"+
			"任务：\n%s\n\n"+
			"返回格式（严格 JSON，不要包含 markdown 标记）：\n"+
			"{\"code\": \"提取到的验证码或空字符串\", \"group\": \"分组名称或空字符串\"}\n\n"+
			"通知消息：\n%s",
		strings.Join(tasks, "\n"), content)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &generationConfig{
			ResponseMimeType: "application/json",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Errorf("Gemini: failed to marshal request: %v", err)
		return AnalysisResult{}
	}

	apiURL := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, geminiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		logger.Errorf("Gemini: failed to create request: %v", err)
		return AnalysisResult{}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("Gemini: request failed: %v", err)
		return AnalysisResult{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Errorf("Gemini: API returned status %d: %s", resp.StatusCode, string(respBody))
		return AnalysisResult{}
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		logger.Errorf("Gemini: failed to decode response: %v", err)
		return AnalysisResult{}
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return AnalysisResult{}
	}

	raw := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		logger.Errorf("Gemini: failed to parse JSON result: %v, raw: %s", err, raw)
		return AnalysisResult{}
	}

	// Validate group value
	if result.Group != "" {
		validGroups := map[string]bool{
			"重要紧急":   true,
			"重要不紧急":  true,
			"紧急不重要":  true,
			"不重要不紧急": true,
		}
		if !validGroups[result.Group] {
			logger.Warnf("Gemini: unexpected group value: %s", result.Group)
			result.Group = ""
		}
	}

	// Clean up code value
	if strings.EqualFold(result.Code, "NONE") || strings.EqualFold(result.Code, "无") {
		result.Code = ""
	}

	if result.Code != "" {
		logger.Infof("Gemini: extracted verification code from message")
	}
	if result.Group != "" {
		logger.Infof("Gemini: notification categorized as [%s]", result.Group)
	}

	return result
}
