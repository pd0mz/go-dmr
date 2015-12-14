#!/bin/bash
#
# sox converts big endian float32 to signed pcm wave

sox --endian big -t f32 - -t wav - | lame -b 96 - -
