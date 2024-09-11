package familytree

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CREATE TABLE familytree (
// 	id BIGINT PRIMARY KEY NOT NULL,
// 	name TEXT NOT NULL,
// 	middle_names TEXT[],
// 	surname TEXT,
// 	nicknames TEXT[],
//  sex TEXT,
//  gender TEXT,
//  pronouns TEXT,
//  dob BIGINT,
//  dod BIGINT,
//  parents BIGINT[],
//  step_parents BIGINT[],
//  guardians BIGINT[],
//  is_adopted BOOLEAN,
//  partner BIGINT,
//  prev_partners BIGINT[]
// );

type Person struct {
	ID           int64    `json:"id" db:"id"`
	Name         string   `json:"name" db:"name"`
	MiddleNames  []string `json:"middle_names" db:"middle_names"`
	Surname      string   `json:"surname" db:"surname"`
	Nicknames    []string `json:"nicknames" db:"nicknames"`
	Sex          string   `json:"sex" db:"sex"`
	Gender       string   `json:"gender" db:"gender"`
	Pronouns     string   `json:"pronouns" db:"pronouns"`
	DOB          int64    `json:"dob" db:"dob"`
	DOD          int64    `json:"dod" db:"dod"`
	Parents      []int64  `json:"parents" db:"parents"`
	StepParents  []int64  `json:"step_parents" db:"step_parents"`
	Guardians    []int64  `json:"guardians" db:"guardians"`
	IsAdopted    bool     `json:"is_adopted" db:"is_adopted"`
	Partner      int64    `json:"partner" db:"partner"`
	PrevPartners []int64  `json:"prev_partners" db:"prev_partners"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) GetPerson(id int64) (*Person, error) {
	var person *Person

	rows, err := s.db.Query(context.Background(), "SELECT * FROM sessions WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	person, err = pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Person])
	if err != nil {
		return nil, err
	}
	return person, nil
}

func (s *Store) GetPersonByName(name string) (*Person, error) {
	var person *Person

	rows, err := s.db.Query(context.Background(), "SELECT * FROM sessions WHERE name = $1", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	person, err = pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Person])
	if err != nil {
		return nil, err
	}
	return person, nil
}

func (s *Store) GetPersonByFullName(name, middleName, surname string) (*Person, error) {
	var person *Person

	rows, err := s.db.Query(context.Background(),
		"SELECT * FROM sessions WHERE name = $1 AND $2 IN (middle_names) AND surname = $3",
		name, middleName, surname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	person, err = pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Person])
	if err != nil {
		return nil, err
	}
	return person, nil
}

func (s *Store) CreatePerson(person *Person) error {
	_, err := s.db.Exec(context.Background(),
		"INSERT INTO familytree (id, name, middle_names, surname, nicknames, sex, gender, pronouns, dob, dod, parents, step_parents, guardians, is_adopted, partner, prev_partners) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)",
		person.ID, person.Name, person.MiddleNames, person.Surname, person.Nicknames, person.Sex, person.Gender, person.Pronouns, person.DOB, person.DOD, person.Parents, person.StepParents, person.Guardians, person.IsAdopted, person.Partner, person.PrevPartners)
	if err != nil {
		return err
	}
	return nil
}
