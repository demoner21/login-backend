package middleware

import "net/http"

// SecurityHeaders adiciona cabeçalhos HTTP de segurança em todas as respostas
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. X-Frame-Options: DENY
		// Impede que seu site seja carregado dentro de um <iframe>.
		// Protege contra Clickjacking (ex: site falso transparente sobre o seu).
		w.Header().Set("X-Frame-Options", "DENY")

		// 2. X-Content-Type-Options: nosniff
		// Força o navegador a respeitar o Content-Type declarado.
		// Impede que arquivos de texto/imagem sejam executados como JS malicioso.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// 3. X-XSS-Protection: 1; mode=block
		// Ativa filtros de XSS antigos em navegadores legacy.
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// 4. Referrer-Policy: strict-origin-when-cross-origin
		// Controla quanta informação de referência (URL anterior) é enviada ao clicar em links.
		// Protege a privacidade do usuário e dados sensíveis na URL.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 5. Content-Security-Policy (CSP)
		// Define de onde o navegador pode carregar recursos (scripts, imagens, fontes).
		// Esta é uma configuração básica que permite Swagger e scripts próprios ('self').
		// Se usar CDNs ou Google Analytics no futuro, precisará ajustar aqui.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")

		// 6. Strict-Transport-Security (HSTS)
		// ⚠️ CUIDADO: Habilite isso apenas em PRODUÇÃO (com HTTPS válido).
		// Se ativado em localhost sem HTTPS, pode bloquear seu acesso ao próprio navegador.
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		next.ServeHTTP(w, r)
	})
}
