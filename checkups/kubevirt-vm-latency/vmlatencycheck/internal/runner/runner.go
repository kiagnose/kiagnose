package runner

import "fmt"

type checkup interface {
	Preflights() error
	Setup() error
	Run() error
	Teardown() error
	Results() map[string]string
}

type reporter interface {
	Report(map[string]string) error
}

type runner struct {
	checkup  checkup
	reporter reporter
}

func New(checkup checkup, reporter reporter) runner {
	return runner{
		checkup:  checkup,
		reporter: reporter,
	}
}

func (r *runner) Run() (runErr error) {
	status := newStatus()
	defer func() {
		status.setResults(r.checkup.Results())
		if err := r.reporter.Report(status.StringsMap()); err != nil {
			status.appendFailureReason(fmt.Errorf("report: %v", err))
		}
		runErr = status.FailureReason()
	}()

	if err := r.checkup.Preflights(); err != nil {
		status.appendFailureReason(fmt.Errorf("preflights: %v", err))
		return err
	}

	if err := r.checkup.Setup(); err != nil {
		status.appendFailureReason(fmt.Errorf("setup: %v", err))
		return err
	}
	defer func() {
		if err := r.checkup.Teardown(); err != nil {
			status.appendFailureReason(fmt.Errorf("teardown: %v", err))
		}
	}()

	if err := r.checkup.Run(); err != nil {
		status.appendFailureReason(fmt.Errorf("run: %v", err))
		return err
	}

	return nil
}
