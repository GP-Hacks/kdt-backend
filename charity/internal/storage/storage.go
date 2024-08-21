package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Collection struct {
	ID           int
	Category     string
	Name         string
	Description  string
	Organization string
	Phone        string
	Website      string
	Goal         int
	Current      int
}

type PostgresStorage struct {
	DB *pgxpool.Pool
}

func NewPostgresStorage(storagePath string) (*PostgresStorage, error) {
	const op = "storage.postgresql.New"
	dbpool, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &PostgresStorage{DB: dbpool}, nil
}

func (s *PostgresStorage) Close() {
	s.DB.Close()
}

func (s *PostgresStorage) GetCategories(ctx context.Context) ([]string, error) {
	const op = "storage.postgresql.GetCategories"
	rows, err := s.DB.Query(ctx, "SELECT DISTINCT category FROM charity")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return categories, nil
}

func (s *PostgresStorage) GetCollections(ctx context.Context) ([]*Collection, error) {
	const op = "storage.postgresql.GetCollections"
	query := "SELECT id, category, name, description, organization, phone, website, goal, current FROM charity"
	return s.fetchPlaces(ctx, query)
}

func (s *PostgresStorage) GetCollectionsByCategory(ctx context.Context, category string) ([]*Collection, error) {
	const op = "storage.postgresql.GetCollectionsByCategory"
	query := "SELECT id, category, name, description, organization, phone, website, goal, current FROM charity WHERE category = $1"
	return s.fetchPlaces(ctx, query, category)
}

func (s *PostgresStorage) fetchPlaces(ctx context.Context, query string, args ...interface{}) ([]*Collection, error) {
	const op = "storage.postgresql.fetchPlaces"
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var collections []*Collection
	for rows.Next() {
		collection := &Collection{}
		err := rows.Scan(
			&collection.ID, &collection.Category, &collection.Name, &collection.Description, &collection.Organization,
			&collection.Phone, &collection.Website, &collection.Goal, &collection.Current,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		collections = append(collections, collection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(collections) == 0 && len(args) > 0 {
		return nil, pgx.ErrNoRows
	}
	return collections, nil
}

func (s *PostgresStorage) FetchAndStoreData(ctx context.Context) error {
	const op = "storage.postgresql.FetchAndStoreData"

	var count int
	err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM charity`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 0 {
		fmt.Println("Data already exists in the database. Skipping fetch and store.")
		return nil
	}

	collections := []Collection{
		{1, "Здравоохранение и медицинская помощь", "Помощь людям больных остеогенезом", " «Хрупкие люди» — это команда единомышленников, неравнодушных к проблемам больных несовершенным остеогенезом. Каждый день нашей работы направлен на открытие новых возможностей для улучшения здоровья людей с врожденной хрупкостью костей и создание условий для их полноценной жизни в обществе.", "Хрупкие люди", "+79035900400", "https://nuzhnapomosh.ru/funds/khrupkie_lyudi_1147799018454/", 6350000, 6328299},
		{
			2, "Социальные услуги", "Помощь жителям Татарстана имеющих трудные жизненные обстоятельства", "Наша миссия заключается не только в очевидном спасении нуждающихся, но и в облагораживании внутреннего мира многих людей. Помогая другим в трудных жизненных обстоятельствах, мы духовно и нравственно растем, а значит, и качество жизни общества со временем улучшается, а уровень безопасности внутри страны растет. На нашем сайте, освещающем нашу работу в Казани, вы можете убедиться, что благотворительность — это не просто слова, а реальная помощь. Наш благотворительный фонд — это не закрытая организация, деятельность которой нужно держать в секрете. Мы принимаем помощь других и всегда готовы представить отчет о своих действиях. ", "Добро даром", "+79370090960", "https://dobrodarom.ru", 800000, 580000,
		},
		{
			3, "Образование и обучение", "Поддержка сельских школ Татарстана", "Сбор средств на обеспечение сельских школ Татарстана современными учебными материалами и оборудованием для обеспечения качественного образования.", "Благотворительный фонд «Школьное будущее» ", "+78435550001", "http://schoolfuture-tatarstan.ru", 2000000, 1250000,
		},
		{
			4, "Здравоохранение и медицинская помощь", "Лечение детей с редкими болезнями", "Сбор средств на лечение детей с редкими генетическими заболеваниями, на покупку медикаментов и проведение необходимых операций.", "Фонд помощи детям «Солнечный свет»", "+78431234567", "http://sunlightfund.ru", 8000000, 3460000,
		},
		{
			5, "Социальные услуги", "Поддержка семьям, находящимся в трудной жизненной ситуации", "Сбор средств для оказания материальной и психологической поддержки семьям, оказавшимся в сложной жизненной ситуации.", "Социальная служба «Надежда и Опора»", "+78436789012", "http://nadezhda-opora-tatarstan.ru", 1500000, 850000,
		},
		{
			6, "Защита окружающей среды и животного мира", "Сохранение редких видов флоры и фауны Татарстана", "Сбор средств на проекты по защите и сохранению редких видов растений и животных на территории Татарстана.", "Экологический фонд «Зеленая Республика» ", "+78433334455", "http://greenrepublicfund.ru", 4000000, 1980000,
		},
		{
			7, "Культура и искусство", "Поддержка молодым артистам Татарстана", "Сбор средств для организации конкурсов, фестивалей и мастер-классов для молодых талантов в области музыки, театра и изобразительного искусства.", "Культурный фонд «Молодые таланты» ", "+78438887766", "http://youngtalents-tatarstan.ru", 3000000, 1530000,
		},
		{
			8, "Образование и обучение", "Программа Стипендий для Студентов из Малообеспеченных Семей", "Сбор средств на предоставление образовательных грантов для студентов из малообеспеченных семей.", "Фонд «Образование для всех»", "+74951234567", "http://educationforall.org", 5000000, 2750000,
		},
		{
			9, "Здравоохранение и медицинская помощь", "Помощь детям с онкологическими заболеваниями", "Сбор средств на лечение детей с онкологическими заболеваниями в ведущих клиниках страны и за рубежом.", "Благотворительный фонд «Надежда»", "+7812987653", "http://hopefund.org", 10000000, 4300000,
		},
		{
			10, "Социальные услуги", "Помощь бездомным", "Сбор средств на обеспечение едой, одеждой и временным жильем для бездомных людей.", "Фонд «Дорога к дому»", "+74991112233", "http://roadtohome.org", 3500000, 1950000,
		},
		{
			11, "Защита окружающей среды и животного мира", "Спасение амурского тигра", "Сбор средств на охрану амурских тигров, их среды обитания и борьбу с браконьерами.", "Фонд охраны дикой природы", "+742177778899", "http://wildlifefund.org", 6000000, 2870000,
		},
		{
			12, "Помощь пострадавшим", "Фонд помощи Курску ‘Вместе сильнее’", "Фонд ‘Вместе сильнее’ занимается сбором средств для оказания помощи пострадавшим в результате недавнего стихийного бедствия в Курске. Собранные средства будут направлены на обеспечение пищи, воды, медикаментов и временного жилья для пострадавших семей.", "Благотворительный фонд ‘Вместе сильнее’", "+74951234567", "http://www.vmestesilnee.ru", 10000000, 2500000,
		},
		{
			13, "Культура и искусство", "Реконструкция исторического музея", "Сбор средств на реконструкцию старинного музейного здания в центре города.", "Ассоциация Культурного Наследия", "+74952233344", "http://culturalheritage.org", 12000000, 5620000,
		},
		{
			14, "Катастрофы и чрезвычайные ситуации", "Помощь пострадавшим от землетрясения в Турции", "Сбор средств на оказание помощи пострадавшим от разрушительного землетрясения в Турции: обеспечение медикаментами, едой и временным жильем.", "Врачи без границ", "+33140212929", "http://msf.org", 1500000000, 790000000,
		},
	}

	for _, collection := range collections {
		_, err := s.DB.Exec(ctx, `
			INSERT INTO charity (category, name, description, organization, phone, website, goal, current)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			collection.Category, collection.Name, collection.Description, collection.Organization, collection.Phone, collection.Website, collection.Goal, collection.Current)
		if err != nil {
			return fmt.Errorf("failed to insert data into table: %w", err)
		}
	}

	return nil
}
