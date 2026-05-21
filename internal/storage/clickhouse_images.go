package storage

import (
	"context"
	"fmt"
)

// InsertImages batch inserts image rows.
func (s *Store) InsertImages(ctx context.Context, images []ImageRow) error {
	if len(images) == 0 {
		return nil
	}

	batch, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO crawlobserver.images (
			crawl_session_id, page_url, image_src, alt, has_alt,
			width, height, crawled_at
		)`)
	if err != nil {
		return fmt.Errorf("preparing images batch: %w", err)
	}

	for _, img := range images {
		if err := batch.Append(
			img.CrawlSessionID, img.PageURL, img.ImageSrc, img.Alt, img.HasAlt,
			img.Width, img.Height, img.CrawledAt,
		); err != nil {
			return fmt.Errorf("appending image row: %w", err)
		}
	}

	return batch.Send()
}

// ListImagesAggregated retrieves unique images grouped by image_src, with
// per-image usage and no-alt counts plus a sample alt and page URL.
func (s *Store) ListImagesAggregated(ctx context.Context, sessionID string, limit, offset int, filters []ParsedFilter, sort *SortParam) ([]ImageAggregateRow, error) {
	query := `
		SELECT
			image_src,
			count() AS usage_count,
			countIf(has_alt = false) AS no_alt_count,
			any(alt) AS sample_alt,
			any(page_url) AS sample_page_url
		FROM crawlobserver.images
		WHERE crawl_session_id = ?`
	args := []interface{}{sessionID}

	whereExtra, filterArgs, err := BuildWhereClause(filters)
	if err != nil {
		return nil, fmt.Errorf("building filter clause: %w", err)
	}
	if whereExtra != "" {
		query += " AND " + whereExtra
		args = append(args, filterArgs...)
	}

	query += " GROUP BY image_src"
	query += BuildOrderByClause(sort, "usage_count DESC, image_src") + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying aggregated images: %w", err)
	}
	defer rows.Close()

	var out []ImageAggregateRow
	for rows.Next() {
		var r ImageAggregateRow
		if err := rows.Scan(
			&r.ImageSrc, &r.UsageCount, &r.NoAltCount, &r.SampleAlt, &r.SamplePageURL,
		); err != nil {
			return nil, fmt.Errorf("scanning aggregated image: %w", err)
		}
		out = append(out, r)
	}
	return out, nil
}

// ListImages retrieves images for a session with pagination, filters, and sort.
func (s *Store) ListImages(ctx context.Context, sessionID string, limit, offset int, filters []ParsedFilter, sort *SortParam) ([]ImageRow, error) {
	query := `
		SELECT crawl_session_id, page_url, image_src, alt, has_alt, width, height, crawled_at
		FROM crawlobserver.images
		WHERE crawl_session_id = ?`
	args := []interface{}{sessionID}

	whereExtra, filterArgs, err := BuildWhereClause(filters)
	if err != nil {
		return nil, fmt.Errorf("building filter clause: %w", err)
	}
	if whereExtra != "" {
		query += " AND " + whereExtra
		args = append(args, filterArgs...)
	}

	query += BuildOrderByClause(sort, "page_url, image_src") + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying images: %w", err)
	}
	defer rows.Close()

	var images []ImageRow
	for rows.Next() {
		var img ImageRow
		if err := rows.Scan(
			&img.CrawlSessionID, &img.PageURL, &img.ImageSrc, &img.Alt, &img.HasAlt,
			&img.Width, &img.Height, &img.CrawledAt,
		); err != nil {
			return nil, fmt.Errorf("scanning image: %w", err)
		}
		images = append(images, img)
	}
	return images, nil
}
