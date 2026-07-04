package http

import (
	stdhttp "net/http"
)

type unitDefDTO struct {
	Code      string   `json:"code"`
	Names     []string `json:"names"`
	Dimension string   `json:"dimension"`
	SiUnit    string   `json:"si_unit"`
	SiFactor  float64  `json:"si_factor"`
	SiOffset  float64  `json:"si_offset"`
}

type synonymAliasDTO struct {
	Value  string `json:"value"`
	Lang   string `json:"lang"`
	Status string `json:"status"`
}

type synonymGroupDTO struct {
	Canonical string            `json:"canonical"`
	Etype     string            `json:"etype"`
	Aliases   []synonymAliasDTO `json:"aliases"`
}

func (server *Server) dictionaryUnitsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	const query = `
SELECT code, names, dimension, si_unit, si_factor::float8, si_offset::float8
FROM kg.units
ORDER BY dimension, code`
	rows, err := server.pool.Query(r.Context(), query)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadGateway, "upstream_error", "Upstream error", err.Error())
		return
	}
	defer rows.Close()

	items := make([]unitDefDTO, 0)
	for rows.Next() {
		var item unitDefDTO
		if err := rows.Scan(&item.Code, &item.Names, &item.Dimension, &item.SiUnit, &item.SiFactor, &item.SiOffset); err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "scan_error", "Internal server error", err.Error())
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "scan_error", "Internal server error", err.Error())
		return
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[unitDefDTO]{Items: items})
}

func (server *Server) dictionarySynonymsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	const query = `
SELECT e.canonical_name, e.etype::text,
       array_agg(a.alias ORDER BY a.alias),
       array_agg(a.lang ORDER BY a.alias),
       array_agg(a.status ORDER BY a.alias)
FROM kg.entity_aliases a
JOIN kg.entities e ON e.id = a.entity_id
WHERE e.status = 'active'
  AND lower(btrim(a.alias)) <> lower(btrim(e.canonical_name))
GROUP BY e.canonical_name, e.etype
ORDER BY count(*) DESC, e.canonical_name
LIMIT 100`
	rows, err := server.pool.Query(r.Context(), query)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadGateway, "upstream_error", "Upstream error", err.Error())
		return
	}
	defer rows.Close()

	items := make([]synonymGroupDTO, 0)
	for rows.Next() {
		var (
			canonical string
			etype     string
			aliases   []string
			langs     []string
			statuses  []string
		)
		if err := rows.Scan(&canonical, &etype, &aliases, &langs, &statuses); err != nil {
			writeProblem(w, r, stdhttp.StatusInternalServerError, "scan_error", "Internal server error", err.Error())
			return
		}
		group := synonymGroupDTO{Canonical: canonical, Etype: etype, Aliases: make([]synonymAliasDTO, 0, len(aliases))}
		for index := range aliases {
			group.Aliases = append(group.Aliases, synonymAliasDTO{
				Value:  aliases[index],
				Lang:   stringAt(langs, index),
				Status: stringAt(statuses, index),
			})
		}
		items = append(items, group)
	}
	if err := rows.Err(); err != nil {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "scan_error", "Internal server error", err.Error())
		return
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[synonymGroupDTO]{Items: items})
}

func stringAt(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}
