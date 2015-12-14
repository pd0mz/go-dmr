/*  vi:si:noexpandtab:sw=4:sts=4:ts=4
*/
/*
 * oggfwd    -- Forward an Ogg stream from stdin to an Icecast server
 *		A useful demonstration of the libshout API
 *
 * This program is distributed under the GNU General Public License, version 2.
 * A copy of this license is included with this source.
 *
 * This program is provided "as-is", with no explicit or implied warranties of
 * any kind.
 *
 * Copyright (C) 2003-2006,	J <j@v2v.cc>,
 *				rafael2k <rafael(at)riseup(dot)net>,
 *				Moritz Grimm <gtgbr@gmx.net>
 * Copyright (C) 2015,          Philipp Schafft <lion@lion.leolix.org>
 */
/* thanx to rhatto <rhatto (AT) riseup (DOT) net> and others at submidialogia :-P */

#include <sys/types.h>
#include <sys/param.h>
#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <fcntl.h>

#ifndef NO_UNISTD_H
#  include <unistd.h>
#endif /* no-NO_UNISTD_H */

#include <shout/shout.h>

extern char    *__progname;
extern char    *optarg;
extern int	optind;
extern int	errno;

volatile sig_atomic_t	print_total = 0;
volatile sig_atomic_t	quit = 0;

#define METABUFSIZE 4096
char    *metafilename;
shout_t *shout;

#define BUFFERSIZE	4096

#if defined(__dead)
__dead void
#else
void
#endif /* __dead */
usage(void)
{
	printf("usage: %s "
	       "[-hp] "
#ifdef SHOUT_TLS
	       "[-T {disabled|auto|auto_no_plain|rfc2818|rfc2817}] "
#endif
	       "[-m metadata file] "
	       "[-d description] "
	       "[-g genre] "
	       "[-n name] "
	       "[-u URL]\n"
	       "              address port password mountpoint\n",
	       __progname);
	exit(1);
}


void 
load_metadata()
{
  int i, fh, r;
  char buf[METABUFSIZE], *key, *val;
  enum {state_comment, state_key, state_value, state_unknown} state;

  bzero(buf, METABUFSIZE);

  if (!metafilename) {
    fprintf(stderr, "Please use the -m argument to set the meta file name!\n");
    return;
  }

  fh = open(metafilename, O_RDONLY);

  if (-1==fh) {
    fprintf(stderr, "Error while opening meta file \"%s\": %s\n", metafilename, strerror(errno));
    return;
  }

  r = read(fh, &buf, METABUFSIZE);
  if (-1==r) {
    fprintf(stderr, "Error while reading meta file \"%s\": %s\n", metafilename, strerror(errno));
    close(fh);
    return;
  }

  state = state_unknown;
  key = val = NULL;
  i = 0;

  while (i<METABUFSIZE) {
    switch (buf[i]) {
    case 0:
      /* we're done */
      i = METABUFSIZE;
      break;

    case '\r':
    case '\n':
      if (state_value==state) {
	buf[i] = 0;
	
	if (key && val) {
	  if (0==strcmp("name", key)) {
	    shout_set_name(shout, val);
	  } else if (0==strcmp("genre", key)) {
	    shout_set_genre(shout, val);
	  } else if (0==strcmp("description", key)) {
	    shout_set_description(shout, val);
	  } else if (0==strcmp("url", key)) {
	    shout_set_url(shout, val);
	  }
	}
      }

      state = state_unknown;
      key = NULL;
      val = NULL;
      break;

    case '=':
      if (state_key==state) {
	buf[i] = 0;
	state = state_value;
	val = &buf[i+1];
      }
      break;

    case '#':
      if (state_unknown==state) {
	state = state_comment;
      }
      break;
      
    default:
      if (state_unknown==state) {
	state = state_key;
	key = &buf[i];
      }
    }
    
    i++;
  }

  close(fh);
}

void
sig_handler(int sig)
{
	switch (sig) {
	case SIGHUP:
		print_total = 1;
		break;
	case SIGTERM:
	case SIGINT:
		quit = 1;
		break;
	case SIGUSR1:
	        load_metadata();
		break;	  
	default:
		/* ignore */
		break;
	}
}

void
set_argument_string(char **param, char *opt, char optname)
{
	size_t siz;

	if (*param) {
		fprintf(stderr, "%s: Parameter -%c given multiple times\n",
			__progname, optname);
		usage();
	}

	siz = strlen(opt) + 1;
	if (siz >= MAXPATHLEN) {
		fprintf(stderr, "%s: Argument for parameter -%c too long\n",
			__progname, optname);
		exit(1);
	}

	if ((*param = malloc(siz)) == NULL) {
		fprintf(stderr, "%s: %s\n", __progname, strerror(errno));
		exit(1);
	}

	snprintf(*param, siz, "%s", opt);
}

#ifdef SHOUT_TLS
void
set_tls_mode(int *tls_mode, char *opt, char optname)
{
	if (0==strcasecmp("DISABLED", opt)) {
		*tls_mode = SHOUT_TLS_DISABLED;
	} else if (0==strcasecmp("AUTO", opt)) {
		*tls_mode = SHOUT_TLS_AUTO;
	} else if (0==strcasecmp("AUTO_NO_PLAIN", opt)) {
		*tls_mode = SHOUT_TLS_AUTO_NO_PLAIN;
	} else if (0==strcasecmp("RFC2818", opt)) {
		*tls_mode = SHOUT_TLS_RFC2818;
	} else if (0==strcasecmp("RFC2817", opt)) {
		*tls_mode = SHOUT_TLS_RFC2817;
	} else {
		fprintf(stderr, "%s: Invalid value for -%c.\n",
			__progname, optname);
		usage();
		exit(1);
	}
}
#endif

int
main(int argc, char **argv)
{
	unsigned char	buff[BUFFERSIZE];
	int		ret, ch;
	unsigned int	pFlag;
	char	       *description, *genre, *name, *url;
	size_t		bytes_read = 0;
	unsigned short	port;
	unsigned long long total;
#ifdef SHOUT_TLS
	int             tls_mode = SHOUT_TLS_AUTO;
#endif

	pFlag = 0;
	description = genre = name = url = metafilename = NULL;
	while ((ch = getopt(argc, argv, "d:g:hn:m:pu:T:")) != -1) {
		switch (ch) {
		case 'd':
			set_argument_string(&description, optarg, 'D');
			break;
		case 'g':
			set_argument_string(&genre, optarg, 'g');
			break;
		case 'n':
			set_argument_string(&name, optarg, 'n');
			break;
		case 'm':
     		        set_argument_string(&metafilename, optarg, 'm');
		        break;
		case 'p':
			pFlag = 1;
			break;
		case 'u':
			set_argument_string(&url, optarg, 'u');
			break;
		case 'T':
#ifdef SHOUT_TLS
			set_tls_mode(&tls_mode, optarg, 'T');
			break;
#endif
		case 'h':
		default:
			usage();
		}
	}
	argc -= optind;
	argv += optind;

	if (argc != 4) {
		fprintf(stderr, "%s: Wrong number of arguments\n", __progname);
		usage();
	}

	if ((shout = shout_new()) == NULL) {
		fprintf(stderr, "%s: Could not allocate shout_t\n",
			__progname);
		return (1);
	}

	shout_set_format(shout, SHOUT_FORMAT_OGG);

#ifdef SHOUT_TLS
	if (shout_set_tls(shout, tls_mode) != SHOUTERR_SUCCESS) {
		fprintf(stderr, "%s: Error setting TLS mode: %s\n", __progname,
			shout_get_error(shout));
		return (1);
	}
#endif

	if (shout_set_host(shout, argv[0]) != SHOUTERR_SUCCESS) {
		fprintf(stderr, "%s: Error setting hostname: %s\n", __progname,
			shout_get_error(shout));
		return (1);
	}

	if (sscanf(argv[1], "%hu", &port) != 1) {
		fprintf(stderr, "Invalid port `%s'\n", argv[1]);
		usage();
	}
	if (shout_set_port(shout, port) != SHOUTERR_SUCCESS) {
		fprintf(stderr, "%s: Error setting port: %s\n", __progname,
			shout_get_error(shout));
		return (1);
	}

	if (shout_set_password(shout, argv[2]) != SHOUTERR_SUCCESS) {
		fprintf(stderr, "%s: Error setting password: %s\n", __progname,
			shout_get_error(shout));
		return (1);
	}

	if (shout_set_mount(shout, argv[3]) != SHOUTERR_SUCCESS) {
		fprintf(stderr, "%s: Error setting mount: %s\n", __progname,
			shout_get_error(shout));
		return (1);
	}

	shout_set_public(shout, pFlag);

	if (metafilename)
       	        load_metadata();

	if (description)
		shout_set_description(shout, description);

	if (genre)
		shout_set_genre(shout, genre);

	if (name)
		shout_set_name(shout, name);

	if (url)
		shout_set_url(shout, url);

	signal(SIGUSR1, sig_handler);

	//wait for data before opening connection to server
	bytes_read = fread(buff, 1, sizeof(buff), stdin);

	if (shout_open(shout) == SHOUTERR_SUCCESS) {
		printf("%s: Connected to server\n", __progname);
		total = 0;

		signal(SIGHUP, sig_handler);
		signal(SIGTERM, sig_handler);
		signal(SIGINT, sig_handler);

		while (quit == 0) {
			total += bytes_read;

			if (bytes_read > 0) {
				ret = shout_send(shout, buff, bytes_read);
				if (ret != SHOUTERR_SUCCESS) {
					printf("%s: Send error: %s\n",
					       __progname,
					       shout_get_error(shout));
					quit = 1;
				}
			} else
				quit = 1;

			if (quit) {
				printf("%s: Quitting ...\n", __progname);
				print_total = 1;
			}

			if (print_total) {
				printf("%s: Total bytes read: %llu\n",
				       __progname, total);
				print_total = 0;
			}

			shout_sync(shout);

			bytes_read = fread(buff, 1, sizeof(buff), stdin);
			if (bytes_read != sizeof(buff) && feof(stdin)) {
				quit = 1;
			}
		}
	} else {
		fprintf(stderr, "%s: Error connecting: %s\n", __progname,
		       shout_get_error(shout));
		return (1);
	}

	shout_close(shout);

	return (0);
}
