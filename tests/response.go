// Package tests - набор тестовых данных для основных структур ответов Дзен Мани
package tests

import "github.com/nemirlev/zenapi"

// getTestTransaction создает тестовую транзакцию.
func getTestTransaction() zenapi.Transaction {
	return zenapi.Transaction{
		ID:                  "1",
		Changed:             123456789,
		Created:             123456789,
		User:                1,
		Deleted:             false,
		Hold:                nil,
		IncomeInstrument:    1,
		IncomeAccount:       "acc1",
		Income:              100,
		OutcomeInstrument:   1,
		OutcomeAccount:      "acc2",
		Outcome:             50,
		Tag:                 []string{"tag1"},
		Merchant:            nil,
		Payee:               "payee1",
		OriginalPayee:       "originalPayee1",
		Comment:             "comment",
		Date:                "2023-01-01",
		Mcc:                 nil,
		ReminderMarker:      nil,
		OpIncome:            nil,
		OpIncomeInstrument:  nil,
		OpOutcome:           nil,
		OpOutcomeInstrument: nil,
		Latitude:            nil,
		Longitude:           nil,
		QRCode:              nil,
		Source:              nil,
		IncomeBankID:        nil,
		OutcomeBankID:       nil,
	}
}

// getTestInstrument создает тестовый инструмент.
func getTestInstrument() zenapi.Instrument {
	return zenapi.Instrument{
		ID:         1,
		Changed:    123456789,
		Title:      "USD",
		ShortTitle: "USD",
		Symbol:     "$",
		Rate:       74.5,
	}
}

// getTestCompany создает тестовую компанию.
func getTestCompany() zenapi.Company {
	return zenapi.Company{
		ID:        1,
		Changed:   123456789,
		Title:     "Company 1",
		FullTitle: "Company 1 Full Title",
		Www:       "www.company1.com",
		Country:   1,
	}
}

// getTestUser создает тестового пользователя.
func getTestUser() zenapi.User {
	return zenapi.User{
		ID:       1,
		Changed:  123456789,
		Login:    nil,
		Currency: 1,
		Parent:   nil,
	}
}

// getTestCountry создает тестовую страну.
func getTestCountry() zenapi.Country {
	return zenapi.Country{
		ID:       1,
		Title:    "Country 1",
		Currency: 1,
		Domain:   "country1",
	}
}

// getTestAccount создает тестовый аккаунт.
func getTestAccount() zenapi.Account {
	return zenapi.Account{
		ID:                    "1",
		Changed:               123456789,
		User:                  1,
		Role:                  nil,
		Instrument:            nil,
		Company:               nil,
		Type:                  "checking",
		Title:                 "Test Account",
		SyncID:                []string{"sync1"},
		Balance:               nil,
		StartBalance:          nil,
		CreditLimit:           nil,
		InBalance:             true,
		Savings:               nil,
		EnableCorrection:      true,
		EnableSMS:             true,
		Archive:               false,
		Capitalization:        nil,
		Percent:               nil,
		StartDate:             "2023-01-01",
		EndDateOffset:         nil,
		EndDateOffsetInterval: "month",
		PayoffStep:            nil,
		PayoffInterval:        nil,
	}
}

// getTestTag создает тестовый тег.
func getTestTag() zenapi.Tag {
	return zenapi.Tag{
		ID:            "1",
		Changed:       123456789,
		User:          1,
		Title:         "Test Tag",
		Parent:        nil,
		Icon:          nil,
		Picture:       nil,
		Color:         nil,
		ShowIncome:    true,
		ShowOutcome:   true,
		BudgetIncome:  true,
		BudgetOutcome: true,
		Required:      nil,
	}
}

// getTestMerchant создает тестового мерчанта.
func getTestMerchant() zenapi.Merchant {
	return zenapi.Merchant{
		ID:      "1",
		Changed: 123456789,
		User:    1,
		Title:   "Test Merchant",
	}
}

// getTestReminder создает тестовое напоминание.
func getTestReminder() zenapi.Reminder {
	return zenapi.Reminder{
		ID:                "1",
		Changed:           123456789,
		User:              1,
		IncomeInstrument:  1,
		IncomeAccount:     "acc1",
		Income:            100,
		OutcomeInstrument: 1,
		OutcomeAccount:    "acc2",
		Outcome:           50,
		Tag:               []string{"tag1"},
		Merchant:          nil,
		Payee:             "payee1",
		Comment:           "comment",
		Interval:          nil,
		Step:              nil,
		Points:            []int{0, 2, 4},
		StartDate:         "2023-01-01",
		EndDate:           nil,
		Notify:            true,
	}
}

// getTestReminderMarker создает тестовый маркер напоминания.
func getTestReminderMarker() zenapi.ReminderMarker {
	return zenapi.ReminderMarker{
		ID:                "1",
		Changed:           123456789,
		User:              1,
		IncomeInstrument:  1,
		IncomeAccount:     "acc1",
		Income:            100,
		OutcomeInstrument: 1,
		OutcomeAccount:    "acc2",
		Outcome:           50,
		Tag:               []string{"tag1"},
		Merchant:          nil,
		Payee:             "payee1",
		Comment:           "comment",
		Date:              "2023-01-01",
		Reminder:          "reminder1",
		State:             "planned",
		Notify:            true,
	}
}

// getTestBudget создает тестовый бюджет.
func getTestBudget() zenapi.Budget {
	return zenapi.Budget{
		Changed:     123456789,
		User:        1,
		Tag:         nil,
		Date:        "2023-01-01",
		Income:      1000,
		IncomeLock:  true,
		Outcome:     500,
		OutcomeLock: true,
	}
}

// getTestResponse создает тестовый ответ.
func getTestResponse() *zenapi.Response {
	return &zenapi.Response{
		Instrument:     []zenapi.Instrument{getTestInstrument()},
		Country:        []zenapi.Country{getTestCountry()},
		Company:        []zenapi.Company{getTestCompany()},
		User:           []zenapi.User{getTestUser()},
		Account:        []zenapi.Account{getTestAccount()},
		Tag:            []zenapi.Tag{getTestTag()},
		Merchant:       []zenapi.Merchant{getTestMerchant()},
		Budget:         []zenapi.Budget{getTestBudget()},
		Reminder:       []zenapi.Reminder{getTestReminder()},
		ReminderMarker: []zenapi.ReminderMarker{getTestReminderMarker()},
		Transaction:    []zenapi.Transaction{getTestTransaction()},
	}
}
