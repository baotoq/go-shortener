package event

// Compile-time interface check
var _ Event = ClickMilestoneReached{}

// ClickMilestoneReached is raised when a URL reaches a click milestone.
type ClickMilestoneReached struct {
	Base
	ShortCode  string `json:"short_code"`
	Milestone  int64  `json:"milestone"`
	ClickCount int64  `json:"click_count"`
}

// Milestones defines the milestones that trigger the event.
var Milestones = []int64{100, 500, 1000, 5000, 10000, 50000, 100000}

// NewClickMilestoneReached creates a new ClickMilestoneReached event.
func NewClickMilestoneReached(shortCode string, milestone, clickCount int64) ClickMilestoneReached {
	return ClickMilestoneReached{
		Base:       NewBase(shortCode),
		ShortCode:  shortCode,
		Milestone:  milestone,
		ClickCount: clickCount,
	}
}

// EventName returns the event name.
func (e ClickMilestoneReached) EventName() string {
	return "url.milestone_reached"
}

// CheckMilestone checks if the click count has reached a new milestone.
// Returns the milestone value if reached, or 0 if no milestone was reached.
func CheckMilestone(previousCount, currentCount int64) int64 {
	for _, milestone := range Milestones {
		if previousCount < milestone && currentCount >= milestone {
			return milestone
		}
	}
	return 0
}
