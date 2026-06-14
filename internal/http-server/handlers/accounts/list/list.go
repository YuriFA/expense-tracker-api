package redirect

import (
	"errors"
	"log/slog"
	"net/http"
)

type AccountsGetter interface {
	GetAccounts() ([]string, error)
}

func New(log *slog.Logger, accountsGetter AccountsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.accounts.list.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		if alias == "" {
			log.Info("alias is empty")
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		aliasURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", slog.String("alias", alias))
			render.JSON(w, r, resp.Error("not found"))
			return
		}

		if err != nil {
			log.Error("failed to get URL", logger.Error(err))
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		http.Redirect(w, r, aliasURL, http.StatusFound)
	}
}
