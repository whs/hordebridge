package worker

type Config struct {
	BridgeAgent         string   `default:"HordeBridge:1.0:https://github.com/whs/hordebridge"`
	HordeServer         string   `default:"https://stablehorde.net/api/" help:"AI Horde server"`
	HordeAPIKey         string   `required:""`
	PriorityUsernames   []string `help:"List of users who have priority with this worker"`
	NSFW                bool     `help:"Allow NSFW generation" default:"true"`
	RequireUpfrontKudos bool     `help:"Only pick up requests where the owner has the required kudos to consume already available"`
	ExtraSlowWorker     bool     `help:"Extra slow workers are excluded from normal requests but users can opt in to use them. Only use when MPS/s < 0.1"`
	HordeModel          string   `help:"Model name to be reported to AI Horde" required:""`
	WorkerName          string   `help:"Name of the worker" required:""`

	OpenaiServer string `help:"OpenAI server" required:""`
	OpenaiAPIKey string `help:"OpenAI API Key"`
	OpenaiModel  string `help:"Model to serve. Must support text completion (not chat)" required:""`

	MaxLength int `help:"Maximum output length in tokens" required:""`
	// MaxContextLength is the maximum input length in tokens
	MaxContextLength int `help:"Maximum input length in tokens" required:""`
}
