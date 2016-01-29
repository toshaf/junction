package junction_test

import (
	"github.com/toshaf/junction"
	"testing"
)

type Person struct {
	Name string
	Age  int
}

func (p *Person) SetName(name string) {
	p.Name = name
}

func Test_Junction_rejects_non_pointer_model(t *testing.T) {
	var out chan Person
	err := junction.Validate(&out, []junction.Source{{
		Input:  make(chan string),
		Model:  Person{}, // this won't fly
		Update: (*Person).SetName,
	}})

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func Test_Run_rejects_non_pointer_channel(t *testing.T) {
	var out chan Person
	err := junction.Validate(out, nil)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func Test_Run_rejects_pointer_to_non_chan(t *testing.T) {
	var out string
	err := junction.Validate(&out, nil)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func Test_Run_creates_channel(t *testing.T) {
	var out chan Person
	junction.New(&out, nil)
	if out == nil {
		t.Error("Channel is nil")
	}
}

func Test_update_via_member_func_yields_updated_value(t *testing.T) {
	s := make(chan string)
	var out chan Person

	junction.New(&out, []junction.Source{{
		Input:  s,
		Update: (*Person).SetName,
		Model:  &Person{Name: "Jeff", Age: 56},
	}})

	s <- "Fishy Bob"

	person := <-out

	fb := Person{Age: 56, Name: "Fishy Bob"}
	if person != fb {
		t.Errorf("Where's Fishy Bob?")
	}
}

func UpdatePersonAge(p *Person, age int) {
	p.Age = age
}

func Test_update_via_non_member_func_yields_updated_value(t *testing.T) {
	i := make(chan int)
	var out chan Person

	junction.New(&out, []junction.Source{{
		Input:  i,
		Update: UpdatePersonAge,
		Model:  &Person{Name: "Jeff", Age: 56},
	}})

	i <- 57

	person := <-out

	older := Person{Age: 57, Name: "Jeff"}

	if person != older {
		t.Errorf("Expected %v but got %v", older, person)
	}
}

type People map[int]*Person

func (people People) ById(ider Ider) (*Person, bool) {
	p, ok := people[ider.Id()]
	return p, ok
}

type Ider interface {
	Id() int
}

type NameUpdate struct {
	id   int
	name string
}

func (u NameUpdate) Id() int {
	return u.id
}

func ApplyNameUpdate(p *Person, u NameUpdate) {
	p.Name = u.name
}

type AgeUpdate struct {
	id int
}

func (u AgeUpdate) Id() int {
	return u.id
}

func ApplyAgeUpdate(p *Person, u AgeUpdate) {
	p.Age++
}

func Test_Model_lookup_function(t *testing.T) {
	people := People{
		123: &Person{Name: "Ann", Age: 23},
		456: &Person{Name: "Bob", Age: 21},
	}

	names := make(chan NameUpdate)
	ages := make(chan AgeUpdate)

	var out chan Person

	junction.New(&out, []junction.Source{
		{
			Input:  names,
			Update: ApplyNameUpdate,
			Model:  people.ById,
		},
		{
			Input:  ages,
			Update: ApplyAgeUpdate,
			Model:  people.ById,
		},
	})

	names <- NameUpdate{
		id:   123,
		name: "Anne",
	}

	anne := <-out

	if anne != (Person{Name: "Anne", Age: 23}) {
		t.Error("Name update failed")
	}

	ages <- AgeUpdate{id: 123}

	anne = <-out

	if anne.Age != 24 {
		t.Error("Age update failed")
	}
	if anne.Name != "Anne" {
		t.Error("Name update was lost")
	}

	ages <- AgeUpdate{id: 456}

	bob := <-out

	if bob != (Person{Name: "Bob", Age: 22}) {
		t.Error("Age update failed")
	}

	// check state
	if *people[123] != (Person{Name: "Anne", Age: 24}) {
		t.Errorf("Wrong value for Anne: %v", people[123])
	}
	if *people[456] != (Person{Name: "Bob", Age: 22}) {
		t.Errorf("Wrong value for Bob: %v", people[456])
	}
}
