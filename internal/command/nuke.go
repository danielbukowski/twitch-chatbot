package command

import (
	"context"
	"fmt"
	"strconv"
)

func ValidateNukeArguments() Filter {
	return func(cb Handler) Handler {
		return func(ctx context.Context, args []string, chatClient chatClient) error {
			if len(args) != 3 {
				return fmt.Errorf("the number of arguments should be equal to 3, got %d)", len(args))
			}

			duration := args[1]
			unit := args[2]

			switch unit {
			case "s", "m", "h", "d":
				break
			default:
				return fmt.Errorf("time unit should be equal to: s, m, h or d")
			}

			d, err := strconv.Atoi(duration)
			if err != nil {
				return fmt.Errorf("duration should be a number")
			}

			if d <= 0 {
				return fmt.Errorf("duration should be positive")
			}

			return cb(ctx, args, chatClient)
		}
	}
}
