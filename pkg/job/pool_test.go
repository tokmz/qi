package job

import (
	"testing"
	"time"
)

// TestReleaseJob_ZeroValue 测试 ReleaseJob 是否正确重置所有字段为零值
func TestReleaseJob_ZeroValue(t *testing.T) {
	// 创建一个包含所有字段的 Job
	now := time.Now()
	job := &Job{
		ID:          "test-id",
		Name:        "test-name",
		Description: "test-description",
		Cron:        "* * * * * *",
		Type:        JobTypeCron,
		Status:      JobStatusRunning,
		HandlerName: "test-handler",
		Payload:     `{"key": "value"}`,
		NextRunAt:   &now,
		LastRunAt:   &now,
		LastResult:  "success",
		RetryCount:  3,
		MaxRetry:    5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 释放到对象池
	ReleaseJob(job)

	// 验证所有字段都被重置为零值
	if job.ID != "" {
		t.Errorf("ID not reset: got %q, want empty string", job.ID)
	}
	if job.Name != "" {
		t.Errorf("Name not reset: got %q, want empty string", job.Name)
	}
	if job.Description != "" {
		t.Errorf("Description not reset: got %q, want empty string", job.Description)
	}
	if job.Cron != "" {
		t.Errorf("Cron not reset: got %q, want empty string", job.Cron)
	}
	if job.Type != "" {
		t.Errorf("Type not reset: got %q, want empty string", job.Type)
	}
	if job.Status != "" {
		t.Errorf("Status not reset: got %q, want empty string", job.Status)
	}
	if job.HandlerName != "" {
		t.Errorf("HandlerName not reset: got %q, want empty string", job.HandlerName)
	}
	if job.Payload != "" {
		t.Errorf("Payload not reset: got %q, want empty string", job.Payload)
	}
	if job.NextRunAt != nil {
		t.Errorf("NextRunAt not reset: got %v, want nil", job.NextRunAt)
	}
	if job.LastRunAt != nil {
		t.Errorf("LastRunAt not reset: got %v, want nil", job.LastRunAt)
	}
	if job.LastResult != "" {
		t.Errorf("LastResult not reset: got %q, want empty string", job.LastResult)
	}
	if job.RetryCount != 0 {
		t.Errorf("RetryCount not reset: got %d, want 0", job.RetryCount)
	}
	if job.MaxRetry != 0 {
		t.Errorf("MaxRetry not reset: got %d, want 0", job.MaxRetry)
	}
	if !job.CreatedAt.IsZero() {
		t.Errorf("CreatedAt not reset: got %v, want zero time", job.CreatedAt)
	}
	if !job.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt not reset: got %v, want zero time", job.UpdatedAt)
	}
}

// TestReleaseRun_ZeroValue 测试 ReleaseRun 是否正确重置所有字段为零值
func TestReleaseRun_ZeroValue(t *testing.T) {
	// 创建一个包含所有字段的 Run
	now := time.Now()
	run := &Run{
		ID:        "test-run-id",
		JobID:     "test-job-id",
		Status:    RunStatusSuccess,
		StartAt:   now,
		EndAt:     &now,
		Output:    "test output",
		Error:     "test error",
		Duration:  1000,
		TraceID:   "test-trace-id",
		CreatedAt: now,
	}

	// 释放到对象池
	ReleaseRun(run)

	// 验证所有字段都被重置为零值
	if run.ID != "" {
		t.Errorf("ID not reset: got %q, want empty string", run.ID)
	}
	if run.JobID != "" {
		t.Errorf("JobID not reset: got %q, want empty string", run.JobID)
	}
	if run.Status != "" {
		t.Errorf("Status not reset: got %q, want empty string", run.Status)
	}
	if !run.StartAt.IsZero() {
		t.Errorf("StartAt not reset: got %v, want zero time", run.StartAt)
	}
	if run.EndAt != nil {
		t.Errorf("EndAt not reset: got %v, want nil", run.EndAt)
	}
	if run.Output != "" {
		t.Errorf("Output not reset: got %q, want empty string", run.Output)
	}
	if run.Error != "" {
		t.Errorf("Error not reset: got %q, want empty string", run.Error)
	}
	if run.Duration != 0 {
		t.Errorf("Duration not reset: got %d, want 0", run.Duration)
	}
	if run.TraceID != "" {
		t.Errorf("TraceID not reset: got %q, want empty string", run.TraceID)
	}
	if !run.CreatedAt.IsZero() {
		t.Errorf("CreatedAt not reset: got %v, want zero time", run.CreatedAt)
	}
}

// TestReleaseJob_Nil 测试 ReleaseJob 处理 nil 指针
func TestReleaseJob_Nil(t *testing.T) {
	// 不应该 panic
	ReleaseJob(nil)
}

// TestReleaseRun_Nil 测试 ReleaseRun 处理 nil 指针
func TestReleaseRun_Nil(t *testing.T) {
	// 不应该 panic
	ReleaseRun(nil)
}

// TestJobPool_NoPollution 测试对象池不会被污染
func TestJobPool_NoPollution(t *testing.T) {
	// 创建并释放一个带数据的 Job
	now := time.Now()
	job1 := &Job{
		ID:          "job1",
		Name:        "Job 1",
		Description: "First job",
		Type:        JobTypeCron,
		Status:      JobStatusRunning,
		HandlerName: "handler1",
		Payload:     `{"data": "job1"}`,
		NextRunAt:   &now,
		RetryCount:  3,
		MaxRetry:    5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	ReleaseJob(job1)

	// 从对象池获取一个新的 Job
	job2 := AcquireJob()

	// 验证新获取的 Job 是干净的（所有字段为零值）
	if job2.ID != "" {
		t.Errorf("Pool polluted: ID = %q, want empty", job2.ID)
	}
	if job2.Name != "" {
		t.Errorf("Pool polluted: Name = %q, want empty", job2.Name)
	}
	if job2.Description != "" {
		t.Errorf("Pool polluted: Description = %q, want empty", job2.Description)
	}
	if job2.Type != "" {
		t.Errorf("Pool polluted: Type = %q, want empty", job2.Type)
	}
	if job2.Status != "" {
		t.Errorf("Pool polluted: Status = %q, want empty", job2.Status)
	}
	if job2.HandlerName != "" {
		t.Errorf("Pool polluted: HandlerName = %q, want empty", job2.HandlerName)
	}
	if job2.Payload != "" {
		t.Errorf("Pool polluted: Payload = %q, want empty", job2.Payload)
	}
	if job2.NextRunAt != nil {
		t.Errorf("Pool polluted: NextRunAt = %v, want nil", job2.NextRunAt)
	}
	if job2.RetryCount != 0 {
		t.Errorf("Pool polluted: RetryCount = %d, want 0", job2.RetryCount)
	}
	if job2.MaxRetry != 0 {
		t.Errorf("Pool polluted: MaxRetry = %d, want 0", job2.MaxRetry)
	}
}

// TestRunPool_NoPollution 测试 Run 对象池不会被污染
func TestRunPool_NoPollution(t *testing.T) {
	// 创建并释放一个带数据的 Run
	now := time.Now()
	run1 := &Run{
		ID:        "run1",
		JobID:     "job1",
		Status:    RunStatusSuccess,
		StartAt:   now,
		EndAt:     &now,
		Output:    "output1",
		Error:     "error1",
		Duration:  1000,
		TraceID:   "trace1",
		CreatedAt: now,
	}
	ReleaseRun(run1)

	// 从对象池获取一个新的 Run
	run2 := AcquireRun()

	// 验证新获取的 Run 是干净的
	if run2.ID != "" {
		t.Errorf("Pool polluted: ID = %q, want empty", run2.ID)
	}
	if run2.JobID != "" {
		t.Errorf("Pool polluted: JobID = %q, want empty", run2.JobID)
	}
	if run2.Status != "" {
		t.Errorf("Pool polluted: Status = %q, want empty", run2.Status)
	}
	if !run2.StartAt.IsZero() {
		t.Errorf("Pool polluted: StartAt = %v, want zero", run2.StartAt)
	}
	if run2.EndAt != nil {
		t.Errorf("Pool polluted: EndAt = %v, want nil", run2.EndAt)
	}
	if run2.Output != "" {
		t.Errorf("Pool polluted: Output = %q, want empty", run2.Output)
	}
	if run2.Error != "" {
		t.Errorf("Pool polluted: Error = %q, want empty", run2.Error)
	}
	if run2.Duration != 0 {
		t.Errorf("Pool polluted: Duration = %d, want 0", run2.Duration)
	}
	if run2.TraceID != "" {
		t.Errorf("Pool polluted: TraceID = %q, want empty", run2.TraceID)
	}
}

// BenchmarkReleaseJob_ZeroValue 基准测试：零值初始化
func BenchmarkReleaseJob_ZeroValue(b *testing.B) {
	now := time.Now()
	job := &Job{
		ID:          "test-id",
		Name:        "test-name",
		Description: "test-description",
		Cron:        "* * * * * *",
		Type:        JobTypeCron,
		Status:      JobStatusRunning,
		HandlerName: "test-handler",
		Payload:     `{"key": "value"}`,
		NextRunAt:   &now,
		LastRunAt:   &now,
		LastResult:  "success",
		RetryCount:  3,
		MaxRetry:    5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*job = Job{}
	}
}

// BenchmarkReleaseJob_Manual 基准测试：手动字段重置（旧方法）
func BenchmarkReleaseJob_Manual(b *testing.B) {
	now := time.Now()
	job := &Job{
		ID:          "test-id",
		Name:        "test-name",
		Description: "test-description",
		Cron:        "* * * * * *",
		Type:        JobTypeCron,
		Status:      JobStatusRunning,
		HandlerName: "test-handler",
		Payload:     `{"key": "value"}`,
		NextRunAt:   &now,
		LastRunAt:   &now,
		LastResult:  "success",
		RetryCount:  3,
		MaxRetry:    5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job.ID = ""
		job.Name = ""
		job.Description = ""
		job.Cron = ""
		job.Type = ""
		job.Status = ""
		job.HandlerName = ""
		job.Payload = ""
		job.NextRunAt = nil
		job.LastRunAt = nil
		job.LastResult = ""
		job.RetryCount = 0
		job.MaxRetry = 0
		job.CreatedAt = time.Time{}
		job.UpdatedAt = time.Time{}
	}
}
