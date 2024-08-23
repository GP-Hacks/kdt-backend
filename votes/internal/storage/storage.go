package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Vote struct {
	ID           int
	Category     string
	Name         string
	Description  string
	Organization string
	EndTime      time.Time
	Photo        string
	Options      []string
}

type RateInfo struct {
	ID           int
	Category     string
	Name         string
	Description  string
	Organization string
	EndTime      time.Time
	Photo        string
	Options      []string
	Mid          float64
}

type PetitionInfo struct {
	ID           int
	Category     string
	Name         string
	Description  string
	Organization string
	EndTime      time.Time
	Photo        string
	Options      []string
	Stats        map[string]int32
}

type ChoiceInfo struct {
	ID           int
	Category     string
	Name         string
	Description  string
	Organization string
	EndTime      time.Time
	Photo        string
	Options      []string
	Stats        map[string]int32
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

func (s *PostgresStorage) GetVotes(ctx context.Context) ([]*Vote, error) {
	const op = "storage.postgresql.GetVotes"
	query := `
		SELECT id, category, name, description, organization, photo, end_time 
		FROM votes
	`
	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var votes []*Vote
	for rows.Next() {
		var vote Vote
		if err := rows.Scan(&vote.ID, &vote.Category, &vote.Name, &vote.Description, &vote.Organization, &vote.Photo, &vote.EndTime); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		if vote.Category == "choice" {
			options, err := s.getOptions(ctx, vote.ID)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", op, err)
			}
			vote.Options = options
		} else {
			vote.Options = []string{}
		}

		votes = append(votes, &vote)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return votes, nil
}

func (s *PostgresStorage) GetRateInfo(ctx context.Context, voteId int) (*RateInfo, error) {
	const op = "storage.postgresql.GetRateInfo"
	query := `
		SELECT id, category, name, description, organization, photo, end_time 
		FROM votes 
		WHERE id = $1 AND category = 'rate'
	`
	var rateInfo RateInfo
	err := s.DB.QueryRow(ctx, query, voteId).Scan(
		&rateInfo.ID, &rateInfo.Category, &rateInfo.Name, &rateInfo.Description,
		&rateInfo.Organization, &rateInfo.Photo, &rateInfo.EndTime,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	mid, err := s.calculateAverageRating(ctx, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	rateInfo.Mid = mid
	rateInfo.Options = []string{}

	return &rateInfo, nil
}

func (s *PostgresStorage) GetPetitionInfo(ctx context.Context, voteId int) (*PetitionInfo, error) {
	const op = "storage.postgresql.GetPetitionInfo"
	query := `
		SELECT id, category, name, description, organization, photo, end_time 
		FROM votes 
		WHERE id = $1 AND category = 'petition'
	`
	var petitionInfo PetitionInfo
	err := s.DB.QueryRow(ctx, query, voteId).Scan(
		&petitionInfo.ID, &petitionInfo.Category, &petitionInfo.Name, &petitionInfo.Description,
		&petitionInfo.Organization, &petitionInfo.Photo, &petitionInfo.EndTime,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stats, err := s.calculatePetitionStats(ctx, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	petitionInfo.Stats = stats
	petitionInfo.Options = []string{}

	return &petitionInfo, nil
}

func (s *PostgresStorage) GetChoiceInfo(ctx context.Context, voteId int) (*ChoiceInfo, error) {
	const op = "storage.postgresql.GetChoiceInfo"
	query := `
		SELECT id, category, name, description, organization, photo, end_time 
		FROM votes 
		WHERE id = $1 AND category = 'choice'
	`
	var choiceInfo ChoiceInfo
	err := s.DB.QueryRow(ctx, query, voteId).Scan(
		&choiceInfo.ID, &choiceInfo.Category, &choiceInfo.Name, &choiceInfo.Description,
		&choiceInfo.Organization, &choiceInfo.Photo, &choiceInfo.EndTime,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	options, err := s.getOptions(ctx, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	choiceInfo.Options = options

	stats, err := s.calculateChoiceStats(ctx, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	choiceInfo.Stats = stats

	return &choiceInfo, nil
}

func (s *PostgresStorage) VoteRate(ctx context.Context, token string, voteId int, rating int) error {
	const op = "storage.postgresql.VoteRate"
	query := `
		INSERT INTO rate_results (vote_id, user_token, rate)
		VALUES ($1, $2, $3)
		ON CONFLICT (vote_id, user_token) 
		DO UPDATE SET rate = EXCLUDED.rate
	`
	_, err := s.DB.Exec(ctx, query, voteId, token, rating)
	if err != nil {
		return fmt.Errorf("%s: failed to record vote: %w", op, err)
	}
	return nil
}

func (s *PostgresStorage) VotePetition(ctx context.Context, token string, voteId int, support string) error {
	const op = "storage.postgresql.VotePetition"
	query := `
		INSERT INTO petition_results (vote_id, user_token, support)
		VALUES ($1, $2, $3)
		ON CONFLICT (vote_id, user_token) 
		DO UPDATE SET support = EXCLUDED.support
	`
	_, err := s.DB.Exec(ctx, query, voteId, token, support)
	if err != nil {
		return fmt.Errorf("%s: failed to record vote: %w", op, err)
	}
	return nil
}

func (s *PostgresStorage) VoteChoice(ctx context.Context, token string, voteId int, choice string) error {
	const op = "storage.postgresql.VoteChoice"
	query := `
		INSERT INTO choices_results (vote_id, user_token, choice)
		VALUES ($1, $2, $3)
		ON CONFLICT (vote_id, user_token) 
		DO UPDATE SET choice = EXCLUDED.choice
	`
	_, err := s.DB.Exec(ctx, query, voteId, token, choice)
	if err != nil {
		return fmt.Errorf("%s: failed to record vote: %w", op, err)
	}
	return nil
}

func (s *PostgresStorage) getOptions(ctx context.Context, voteId int) ([]string, error) {
	const op = "storage.postgresql.getOptions"
	query := `
		SELECT option 
		FROM options 
		WHERE vote_id = $1
	`
	rows, err := s.DB.Query(ctx, query, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var options []string
	for rows.Next() {
		var option string
		if err := rows.Scan(&option); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return options, nil
}

func (s *PostgresStorage) calculateAverageRating(ctx context.Context, voteId int) (float64, error) {
	const op = "storage.postgresql.calculateAverageRating"
	query := `
		SELECT COALESCE(AVG(rate), 0) 
		FROM rate_results 
		WHERE vote_id = $1
	`
	var mid float64
	err := s.DB.QueryRow(ctx, query, voteId).Scan(&mid)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return mid, nil
}

func (s *PostgresStorage) calculatePetitionStats(ctx context.Context, voteId int) (map[string]int32, error) {
	const op = "storage.postgresql.calculatePetitionStats"
	query := `
		SELECT support, COUNT(*) 
		FROM petition_results 
		WHERE vote_id = $1 
		GROUP BY support
	`
	rows, err := s.DB.Query(ctx, query, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	stats := make(map[string]int32)
	for rows.Next() {
		var support string
		var count int
		if err := rows.Scan(&support, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		stats[support] = int32(count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return stats, nil
}

func (s *PostgresStorage) calculateChoiceStats(ctx context.Context, voteId int) (map[string]int32, error) {
	const op = "storage.postgresql.calculateChoiceStats"
	query := `
		SELECT choice, COUNT(*) 
		FROM choices_results 
		WHERE vote_id = $1 
		GROUP BY choice
	`
	rows, err := s.DB.Query(ctx, query, voteId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	stats := make(map[string]int32)
	for rows.Next() {
		var choice string
		var count int
		if err := rows.Scan(&choice, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		stats[choice] = int32(count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return stats, nil
}

func (s *PostgresStorage) FetchAndStoreData(ctx context.Context) error {
	const op = "storage.postgresql.FetchAndStoreData"

	var count int
	err := s.DB.QueryRow(ctx, "SELECT COUNT(*) FROM votes").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 0 {
		fmt.Println("Data already exists in the database. Skipping fetch and store.")
		return nil
	}

	votes := []Vote{
		{1, "choice", "Лучший кружок по интересам", "Опрос о том, какой кружок по интересам в вашем районе вы считаете самым интересным и полезным.", "Управление молодежной политики Республики Татарстан", time.Now().Add(154 * time.Hour), "https://krupki.by/images/zastavki/deti_tvorchestvo_2.jpg", []string{"Кружок робототехники", "Художественная студия", "Спортивная секция", "Музыкальная группа"}},
		{2, "choice", "Лучшее место для отдыха в Татарстане", "Опрос о том, какое место для отдыха в Татарстане вы считаете самым привлекательным.", "Министерство туризма Республики Татарстан", time.Now().Add(254 * time.Hour), "https://cdn.tripster.ru/thumbs2/1d8c9102-e90d-11ed-9add-42476a0af5aa.1220x600.jpeg", []string{"Казанская набережная", "Национальный парк «Шульган-Таш»", "Озеро Кабан", "Гора Муслюмово"}},
		{3, "petition", "Создание велодорожек в Казани", "Поддержите петицию о создании велодорожек для безопасного передвижения велосипедистов по городу.", "Группа инициативных граждан", time.Now().Add(204 * time.Hour), "https://sun9-66.userapi.com/impg/0PdgWVSRvBbkcwrwuNbNhTZfU-Tk6S0oPH4cKQ/5awLbsk3B_M.jpg?size=1052x596&quality=95&sign=c1b6b3e55f319113dbd14a8e0fd03ada&type=album", []string{}},
		{4, "petition", "Запрос на улучшение общественного транспорта", "Подпишите петицию за улучшение качества общественного транспорта в нашем районе.", "Общественное движение «Транспорт для всех»", time.Now().Add(554 * time.Hour), "https://kazantransport.ru/information_items_property_761.jpg", []string{}},
		{5, "rate", "Отзыв о работе общественного транспорта", "Поделитесь своим мнением о качестве работы общественного транспорта в вашем районе. Ваши отзывы помогут улучшить сервис.", "Министерство транспорта Республики Татарстан", time.Now().Add(354 * time.Hour), "https://sun9-68.userapi.com/s/v1/ig2/ZcNGIpVANdONHaduKo_AyI_ZGO70gCmsJoERl6ueb2qWLKHp20zyZ0VT1XjRrqjNDCdtNMFiphriuiolRj5PyDls.jpg?quality=95&as=32x24,48x36,72x54,108x81,160x120,240x180,360x270,480x360,540x405,640x480,720x540,870x653&from=bu&u=bAdxtPh4rqpatU9DDn8YeaUbV95ztvCXd3J8ADBTqaQ&cs=807x606", []string{}},
		{6, "rate", "Отзыв о культурном мероприятии", "Поделитесь своим впечатлением о культурном мероприятии, которое вы посетили. Ваши отзывы помогут организовать лучшие события в будущем.", "Управление культуры Республики Татарстан", time.Now().Add(194 * time.Hour), "https://ucare.timepad.ru/a7c550ce-b1a7-4ee2-ab8f-81759077108c/-/preview/600x600/", []string{}},
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, vote := range votes {
		var voteID int
		err := tx.QueryRow(ctx, `
            INSERT INTO votes (category, name, description, organization, photo, end_time)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id`,
			vote.Category, vote.Name, vote.Description, vote.Organization, vote.Photo, vote.EndTime).Scan(&voteID)
		if err != nil {
			return fmt.Errorf("failed to insert vote: %w", err)
		}

		for _, option := range vote.Options {
			_, err = tx.Exec(ctx, `INSERT INTO options (vote_id, option) VALUES ($1, $2)`, voteID, option)
			if err != nil {
				return fmt.Errorf("failed to insert option: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
