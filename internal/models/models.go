package models

import "time"

type PartyMember struct {
	Id              int64  `json:"id"`
	PartyCardNumber string `json:"party_card_number"`
	Name            FIO
	JoiningDate     PartyDate `json:"joining_date"`
	Committee       Committee
	Contribution    []Contribution
	Degree          string
	BirthDate       time.Time `json:"birth_date"`
	Profession      string
	Address         Address
}

type Contribution struct {
	ContributionData PartyDate `json:"contribution_data"`
	sum              int64
}

type PartyDate struct {
	Month int
	Year  int
}

type FIO struct {
	FirstName  string `json:"first_name"`
	FatherName string `json:"father_name"`
	LastName   string `json:"last_name"`
}

type Address struct {
	City     string
	Street   string
	House    int
	Building string
}

type Committee struct {
	DistrictCommittee string `json:"district_committee"`
	RegionalCommittee string `json:"regional_committee"`
}
