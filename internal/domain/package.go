package domain

import (
	"database/sql/driver"
	"errors"
)

// OrderPackager определяет интерфейс для типов упаковки
type OrderPackager interface {
	ValidateWeight(weight float64) error
	GetPackageCost() float64
	Type() string
}

type OrderPackageType struct {
	OrderPackager
}

// Реализация различных типов упаковки
type Default struct{}
type StandardPackage struct{}
type Box struct{}
type Film struct{}

func newDefaultPackage() OrderPackager  { return &Default{} }
func newStandardPackage() OrderPackager { return &StandardPackage{} }
func newBoxPackage() OrderPackager      { return &Box{} }
func newFilmPackage() OrderPackager     { return &Film{} }

// PackageTypeKey тип для определения ключей типов упаковки
type PackageTypeKey string

const (
	DefaultPackageKey PackageTypeKey = "without package"
	PackageKey        PackageTypeKey = "package"
	BoxKey            PackageTypeKey = "box"
	FilmKey           PackageTypeKey = "film"
)

const (
	DefaultType = "without package"
	PackageType = "package"
	BoxType     = "box"
	FilmType    = "film"
)

const (
	DefaultCost float64 = 0
	FilmCost    float64 = 0
	PackageCost float64 = 5
	BoxCost     float64 = 20
)

const (
	PackageMaxWeight float64 = 10
	BoxMaxWeight     float64 = 30
)

// packageConstructors карта для хранения конструкторов типов упаковки
var packageConstructors = map[PackageTypeKey]func() OrderPackager{
	DefaultPackageKey: newDefaultPackage,
	PackageKey:        newStandardPackage,
	BoxKey:            newBoxPackage,
	FilmKey:           newFilmPackage,
}

// toOrderPackageType для преобразования строки в PackageTypeKey
func toOrderPackageType(packageType string) (PackageTypeKey, error) {
	switch packageType {
	case "package":
		return PackageKey, nil
	case "box":
		return BoxKey, nil
	case "film":
		return FilmKey, nil
	case "", "without package":
		return DefaultPackageKey, nil
	default:
		return "", ErrPackageTypeUnsupported
	}
}

// NewPackageType функция для создания нового типа упаковки
func NewPackageType(packageType string) (*OrderPackageType, error) {
	key, err := toOrderPackageType(packageType)
	if err != nil {
		return nil, err
	}

	if constructor, exists := packageConstructors[key]; exists {
		orderPackager := constructor()
		return &OrderPackageType{OrderPackager: orderPackager}, nil
	}

	return nil, ErrPackageTypeUnsupported
}

// Реализация методов OrderPackager для каждого типа упаковки
func (d *Default) ValidateWeight(weight float64) error {
	if weight <= 0 {
		return ErrWeightNegative
	}
	return nil
}
func (d *Default) GetPackageCost() float64 { return DefaultCost }
func (d *Default) Type() string            { return DefaultType }

func (p *StandardPackage) ValidateWeight(weight float64) error {
	if weight <= 0 {
		return ErrWeightNegative
	}
	if weight > PackageMaxWeight {
		return ErrWeightExceedsLimit{
			Weight: weight,
			Limit:  PackageMaxWeight,
		}
	}
	return nil
}
func (p *StandardPackage) GetPackageCost() float64 { return PackageCost }
func (p *StandardPackage) Type() string            { return PackageType }

func (b *Box) ValidateWeight(weight float64) error {
	if weight <= 0 {
		return ErrWeightNegative
	}
	if weight > BoxMaxWeight {
		return ErrWeightExceedsLimit{
			Weight: weight,
			Limit:  BoxMaxWeight,
		}
	}
	return nil
}
func (b *Box) GetPackageCost() float64 { return BoxCost }
func (b *Box) Type() string            { return BoxType }

func (f *Film) ValidateWeight(weight float64) error {
	if weight <= 0 {
		return ErrWeightNegative
	}
	return nil
}
func (f *Film) GetPackageCost() float64 { return FilmCost }
func (f *Film) Type() string            { return FilmType }

// Реализация интерфейсов Valuer и Scanner для OrderPackageType
func (opt OrderPackageType) Value() (driver.Value, error) {
	if opt.OrderPackager == nil {
		return nil, nil
	}
	return opt.OrderPackager.Type(), nil
}

func (opt *OrderPackageType) Scan(src interface{}) error {
	if src == nil {
		opt.OrderPackager = nil
		return nil
	}

	str, ok := src.(string)
	if !ok {
		return errors.New("source is not a string")
	}

	packageType, err := NewPackageType(str)
	if err != nil {
		return err
	}

	*opt = *packageType
	return nil
}
