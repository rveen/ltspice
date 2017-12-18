# LTSpice utilities, in Go

## lta

    lta [-s] [-v] file.raw

Produces summary results, mainly for use in worst case analysis. For each variable
written by LTSpice to the 'raw' file, the following values are calculated:

- Mean
- Standard deviation (unbiased), corrected for the number of runs by the c4(n) factor [1]
- Min, Max: these values are derived from special variables of the form variable_max and variable_min, if present. If min and max are present, the Cpk, % good and ppm columns are calculated.
- Cpk, the process capability: min of ( (Max - Mean) / 3σ, (Mean - Min) / 3σ )
- % good: how many parts are expected to be within the specified tolerances during production and operation.
- ppm: how many parts in a million are expected to be out of the specified tolerances during production and operation.

The last 3 columns are based on the assumption that the values of the variable
follow a normal (Gaussian) distribution.

The RAW file can be in uncompressed LTSpice IV or XVII formats, with single or double
precision data points (.numcfg higher than 6 produces double precision values).

[1] https://en.wikipedia.org/wiki/Unbiased_estimation_of_standard_deviation#Bias_correction
