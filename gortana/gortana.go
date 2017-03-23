package gortana
/*
  This is very lightly modified from:
  https://github.com/xlab/pocketsphinx-go/blob/master/example/gortana/main.go
*/

import (
  "flag"
	"log"
	"os"
	"unsafe"

	"github.com/xlab/closer"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"github.com/xlab/portaudio-go/portaudio"
)

const (
	samplesPerChannel = 512
	sampleRate        = 16000
	channels          = 1
	sampleFormat      = portaudio.PaInt16
)

var (
  gopath  = os.Getenv("GOPATH")
	hmm     = flag.String("hmm", gopath + "src/github.com/cmusphinx/pocketsphinx/model/en-us/en-us", "Sets directory containing acoustic model files.")
	dict    = flag.String("dict", gopath + "src/github.com/cmusphinx/pocketsphinx/model/en-us/cmudict-en-us.dict", "Sets main pronunciation dictionary (lexicon) input file..")
	lm      = flag.String("lm", gopath + "src/github.com/cmusphinx/pocketsphinx/model/en-us/en-us.lm.bin", "Sets word trigram language model input file.")
	logfile = flag.String("log", "gortana.log", "Log file to write log to.")
  debugFile = flag.String("debug", "gortana.debug.log", "Debug log file to write log to.")
	stdout  = flag.Bool("stdout", false, "Disables log file and writes everything to stdout.")
	outraw  = flag.String("outraw", "", "Specify output dir for RAW recorded sound files (s16le). Directory must exist.")

  savedStream *portaudio.Stream
)

func Listen(inform chan string) {
	go listen(inform)
}

func Pause() {
  portaudio.StopStream(savedStream)
}

func Resume() {
  portaudio.StartStream(savedStream)
}

func listen(inform chan string) {
  debug, err := os.OpenFile(*debugFile, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
      closer.Fatalln(err)
  }
  log.SetOutput(debug)

	defer closer.Close()
	closer.Bind(func() {
    inform <- "Bye!"
		log.Println("Bye!")
    debug.Close()
	})
	if err := portaudio.Initialize(); paError(err) {
		log.Fatalln("PortAudio init error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.Terminate(); paError(err) {
			log.Println("PortAudio term error:", paErrorText(err))
		}
	})

	// Init CMUSphinx
	cfg := sphinx.NewConfig(
		sphinx.HMMDirOption(*hmm),
		sphinx.DictFileOption(*dict),
		sphinx.LMFileOption(*lm),
		sphinx.SampleRateOption(sampleRate),
	)
	if len(*outraw) > 0 {
		sphinx.RawLogDirOption(*outraw)(cfg)
	}
	if *stdout == false {
		sphinx.LogFileOption(*logfile)(cfg)
	}

  log.Println(cfg)

	log.Println("Loading CMU PhocketSphinx.")
	log.Println("This may take a while depending on the size of your model.")
	dec, err := sphinx.NewDecoder(cfg)
	if err != nil {
		closer.Fatalln(err)
	}
	closer.Bind(func() {
		dec.Destroy()
	})
	l := &Listener{
    inform: inform,
		dec: dec,
	}

	var stream *portaudio.Stream
	if err := portaudio.OpenDefaultStream(&stream, channels, 0, sampleFormat, sampleRate,
		samplesPerChannel, l.paCallback, nil); paError(err) {
		log.Fatalln("PortAudio error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.CloseStream(stream); paError(err) {
			log.Println("[WARN] PortAudio error:", paErrorText(err))
		}
	})

	if err := portaudio.StartStream(stream); paError(err) {
		log.Fatalln("PortAudio error:", paErrorText(err))
	}
	closer.Bind(func() {
		if err := portaudio.StopStream(stream); paError(err) {
			log.Fatalln("[WARN] PortAudio error:", paErrorText(err))
		}
	})

  savedStream = stream

	if !dec.StartUtt() {
		closer.Fatalln("[ERR] Sphinx failed to start utterance")
	}
	log.Println(banner)
	log.Println("Ready..")
	closer.Hold()
}

type Listener struct {
  inform     chan string
	inSpeech   bool
	uttStarted bool
	dec        *sphinx.Decoder
}

// paCallback: for simplicity reasons we process raw audio with sphinx in the this stream callback,
// never do that for any serious applications, use a buffered channel instead.
func (l *Listener) paCallback(input unsafe.Pointer, _ unsafe.Pointer, sampleCount uint,
	_ *portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags, _ unsafe.Pointer) int32 {

	const (
		statusContinue = int32(portaudio.PaContinue)
		statusAbort    = int32(portaudio.PaAbort)
	)

	in := (*(*[1 << 24]int16)(input))[:int(sampleCount)*channels]
	// ProcessRaw with disabled search because callback needs to be relatime
	_, ok := l.dec.ProcessRaw(in, true, false)
	// log.Printf("processed: %d frames, ok: %v", frames, ok)
	if !ok {
		return statusAbort
	}
	if l.dec.IsInSpeech() {
		l.inSpeech = true
		if !l.uttStarted {
			l.uttStarted = true
			log.Println("Listening..")
		}
	} else if l.uttStarted {
		// speech -> silence transition, time to start new utterance
		l.dec.EndUtt()
		l.uttStarted = false
		l.report() // report results
		if !l.dec.StartUtt() {
			closer.Fatalln("[ERR] Sphinx failed to start utterance")
		}
	}
	return statusContinue
}

func (l *Listener) report() {
	hyp, _ := l.dec.Hypothesis()
	if len(hyp) > 0 {
		log.Printf("    > hypothesis: %s", hyp)
    l.inform <- hyp
		return
	}
	log.Println("ah, nothing")
}

func paError(err portaudio.Error) bool {
	return portaudio.ErrorCode(err) != portaudio.PaNoError
}

func paErrorText(err portaudio.Error) string {
	return portaudio.GetErrorText(err)
}

const banner = `
 __
/ _  _  _|_ _  _  _
\__)(_)| |_(_|| )(_|
`
