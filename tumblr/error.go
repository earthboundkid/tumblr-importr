package tumblr

import "fmt"

func wrap(err error, msg string) error {
	if err == nil {
		return err
	}
	return fmt.Errorf(msg+": %w", err)
}
