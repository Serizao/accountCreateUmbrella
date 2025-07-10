package DnsSRV



import(
	"log"
	"github.com/miekg/dns"
	"net"
)

func StartDns(){
	dns.HandleFunc(".", handleDNSRequest)
	udp := &dns.Server{Addr: ":53", Net: "udp"}
		log.Println("[+] DNS Serveur Started")
		if err := udp.ListenAndServe(); err != nil {
			log.Fatalf("Erreur UDP: %v", err)
		}
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	for _, q := range r.Question {
		log.Printf("[i] Received DNS Call : %s %s", dns.TypeToString[q.Qtype], q.Name)

		switch q.Qtype {
		case dns.TypeMX:
			mx := &dns.MX{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Mx:         "mx.dns.wleberre.fr.",
				Preference: 10,
			}
			msg.Answer = append(msg.Answer, mx)

			// Ajouter un enregistrement A associé
			a := &dns.A{
				Hdr: dns.RR_Header{
					Name:   "mx.dns.wleberre.fr.",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				A: net.ParseIP("144.91.82.35"),
			}
			msg.Extra = append(msg.Extra, a)

		case dns.TypeA:
			// Répond avec A direct si demandé
			a := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				A: net.ParseIP("144.91.82.35"),
			}
			msg.Answer = append(msg.Answer, a)

		default:
			// Ne répond pas pour les autres types
			msg.Rcode = dns.RcodeNameError
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Println("Erreur d'envoi :", err)
	}
}


